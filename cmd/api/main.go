package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Rahmannugar/authlier"
	"github.com/Rahmannugar/authlier/emailverification"
	authlierpostgres "github.com/Rahmannugar/authlier/storage/postgres"
	authlierredis "github.com/Rahmannugar/authlier/storage/redis"
	consumelauthentication "github.com/Rahmannugar/consumel-server/internal/authentication"
	authenticationservices "github.com/Rahmannugar/consumel-server/internal/authentication/services"
	"github.com/Rahmannugar/consumel-server/internal/config"
	"github.com/Rahmannugar/consumel-server/internal/health"
	infraauthentication "github.com/Rahmannugar/consumel-server/internal/infra/authentication"
	"github.com/Rahmannugar/consumel-server/internal/infra/cache"
	"github.com/Rahmannugar/consumel-server/internal/infra/clients"
	"github.com/Rahmannugar/consumel-server/internal/infra/database"
	"github.com/Rahmannugar/consumel-server/internal/infra/ratelimit"
	"github.com/Rahmannugar/consumel-server/internal/infra/telemetry"
	organizationrepositories "github.com/Rahmannugar/consumel-server/internal/organizations/repositories"
	userrepositories "github.com/Rahmannugar/consumel-server/internal/users/repositories"
	userservices "github.com/Rahmannugar/consumel-server/internal/users/services"
	"github.com/gin-gonic/gin"
	"github.com/resend/resend-go/v2"
)

const (
	readHeaderTimeout     = 5 * time.Second
	readTimeout           = 15 * time.Second
	idleTimeout           = 60 * time.Second
	shutdownTimeout       = 15 * time.Second
	databaseTimeout       = 10 * time.Second
	authMigrationTimeout  = 30 * time.Second
	resendTimeout         = 10 * time.Second
	sessionLifetime       = 7 * 24 * time.Hour
	sessionCacheTTL       = time.Hour
	maximumActiveSessions = 3
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(); err != nil {
		logger.Error("api stopped", "error", err)
		os.Exit(1)
	}
}

func run() (runError error) {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	telemetryRuntime, err := telemetry.New(context.Background(), string(cfg.Environment))
	if err != nil {
		return fmt.Errorf("initialize telemetry: %w", err)
	}
	logger := telemetryRuntime.Logger()
	slog.SetDefault(logger)
	defer func() {
		telemetryContext, cancelTelemetry := context.WithTimeout(
			context.Background(),
			shutdownTimeout,
		)
		defer cancelTelemetry()
		if err := telemetryRuntime.Shutdown(telemetryContext); err != nil {
			runError = errors.Join(runError, fmt.Errorf("shutdown telemetry: %w", err))
		}
	}()
	if cfg.Environment != config.EnvironmentDevelopment {
		gin.SetMode(gin.ReleaseMode)
	}

	databaseContext, cancelDatabase := context.WithTimeout(context.Background(), databaseTimeout)
	databasePool, err := database.Open(databaseContext, cfg.Database.URL)
	cancelDatabase()
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer databasePool.Close()

	authlierPostgres, err := authlierpostgres.New(databasePool, authlierpostgres.Config{})
	if err != nil {
		return fmt.Errorf("configure Authlier PostgreSQL storage: %w", err)
	}
	authMigrationContext, cancelAuthMigration := context.WithTimeout(
		context.Background(),
		authMigrationTimeout,
	)
	err = authlierPostgres.Migrate(authMigrationContext)
	cancelAuthMigration()
	if err != nil {
		return fmt.Errorf("migrate Authlier PostgreSQL storage: %w", err)
	}

	redisClient, err := cache.Open(cfg.Redis.URL)
	if err != nil {
		return fmt.Errorf("connect Redis: %w", err)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			logger.Error("close Redis client", "error", err)
		}
	}()

	redisSessionCache, err := authlierredis.NewSessionCache(redisClient, "consumel:auth")
	if err != nil {
		return fmt.Errorf("configure Authlier session cache: %w", err)
	}
	sessionCache, err := infraauthentication.NewJitteredSessionCache(redisSessionCache)
	if err != nil {
		return fmt.Errorf("configure jittered session cache: %w", err)
	}
	emailVerificationStore, err := infraauthentication.NewRedisEmailVerificationStore(
		databasePool,
		redisClient,
		cfg.Auth.OTPHMACSecret,
		"consumel:auth:email-verification",
	)
	if err != nil {
		return fmt.Errorf("configure Redis email verification storage: %w", err)
	}
	authlierDatabase, err := infraauthentication.NewAuthlierDatabase(
		authlierPostgres,
		databasePool,
		sessionCache,
		emailVerificationStore,
		maximumActiveSessions,
		logger,
	)
	if err != nil {
		return fmt.Errorf("configure Consumel Authlier storage policy: %w", err)
	}
	distributedLimiter, err := ratelimit.NewRedisLimiter(
		redisClient,
		cfg.Auth.OTPHMACSecret,
		"consumel:rate-limit",
	)
	if err != nil {
		return fmt.Errorf("configure distributed authentication rate limiter: %w", err)
	}
	passwordAttemptGuard := infraauthentication.NewPasswordAttemptGuard(distributedLimiter)
	otpAttemptGuard := infraauthentication.NewOTPAttemptGuard(distributedLimiter)

	resendClient := resend.NewCustomClient(telemetry.NewHTTPClient(resendTimeout), cfg.Resend.APIKey)
	verificationSender, err := clients.NewResendAuthenticationEmailSender(
		resendClient.Emails,
		cfg.Resend.NoReplyFrom,
	)
	if err != nil {
		return fmt.Errorf("configure authentication email sender: %w", err)
	}

	auth, err := authlier.New(authlier.Config{
		AppName:        "Consumel",
		BaseURL:        cfg.Auth.BaseURL,
		Database:       authlierDatabase,
		TrustedOrigins: cfg.Auth.TrustedOrigins,
		TrustedProxies: cfg.Auth.TrustedProxies,
		EmailAndPassword: authlier.EmailAndPasswordConfig{
			Enabled:                  true,
			RequireEmailVerification: true,
			ValidatePassword:         consumelauthentication.ValidatePassword,
			AttemptGuard:             passwordAttemptGuard,
		},
		EmailVerification: authlier.EmailVerificationConfig{
			Enabled:                     true,
			Delivery:                    emailverification.DeliveryMethodOTP,
			OTPSecret:                   cfg.Auth.OTPHMACSecret,
			Sender:                      verificationSender,
			SendOnSignUp:                true,
			AutoSignInAfterVerification: true,
			AttemptGuard:                otpAttemptGuard,
		},
		Session: authlier.SessionConfig{
			Mode:     authlier.SessionModeCookie,
			Lifetime: sessionLifetime,
			Cache:    sessionCache,
			CacheTTL: sessionCacheTTL,
			Cookie: authlier.CookieConfig{
				Name:     "consumel_session",
				Path:     "/",
				SameSite: http.SameSiteLaxMode,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("configure Authlier: %w", err)
	}

	userRepository := userrepositories.NewUserRepository(databasePool)
	userService := userservices.NewUserService(userRepository)
	organizationRepository := organizationrepositories.NewOrganizationRepository(databasePool)
	tenantService := authenticationservices.NewAuthenticatedTenantService(
		infraauthentication.NewAuthlierSessionResolver(auth),
		userService,
		organizationRepository,
	)
	authenticationHandler := consumelauthentication.NewHandler(tenantService, logger)
	authenticationMux := http.NewServeMux()
	authenticationMux.HandleFunc("GET /api/auth/context", authenticationHandler.Context)
	authenticationMux.Handle("/", auth.Handler())

	router := gin.New()
	requestTelemetry, err := telemetryRuntime.HTTPMiddleware()
	if err != nil {
		return fmt.Errorf("configure HTTP telemetry: %w", err)
	}
	router.Use(requestTelemetry, gin.Recovery())
	if err := router.SetTrustedProxies(cfg.Auth.TrustedProxies); err != nil {
		return fmt.Errorf("configure trusted HTTP proxies: %w", err)
	}
	health.RegisterRoutes(router, databasePool)
	authenticationRoutes := router.Group("/api/auth")
	authenticationRoutes.Use(
		consumelauthentication.RequestTelemetry(),
		consumelauthentication.RateLimitRequests(distributedLimiter, logger),
	)
	authenticationRoutes.Any("/*path", gin.WrapH(authenticationMux))

	server := &http.Server{
		Addr:              cfg.HTTP.Address(),
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		IdleTimeout:       idleTimeout,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("api listening", "address", server.Addr, "environment", cfg.Environment)
		err := server.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serverErrors <- err
	}()

	shutdownSignal, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErrors:
		if err != nil {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	case <-shutdownSignal.Done():
		logger.Info("api shutdown started")
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownContext); err != nil {
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}
	if err := <-serverErrors; err != nil {
		return fmt.Errorf("stop HTTP server: %w", err)
	}

	logger.Info("api shutdown completed")
	return nil
}
