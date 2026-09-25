package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/Rahmannugar/authlier"
	"github.com/Rahmannugar/authlier/emailverification"
	authlierpostgres "github.com/Rahmannugar/authlier/storage/postgres"
	authlierredis "github.com/Rahmannugar/authlier/storage/redis"
	consumelauthentication "github.com/Rahmannugar/consumel-server/internal/authentication"
	authenticationservices "github.com/Rahmannugar/consumel-server/internal/authentication/services"
	"github.com/Rahmannugar/consumel-server/internal/config"
	infraauthentication "github.com/Rahmannugar/consumel-server/internal/infra/authentication"
	"github.com/Rahmannugar/consumel-server/internal/infra/cache"
	"github.com/Rahmannugar/consumel-server/internal/infra/clients"
	"github.com/Rahmannugar/consumel-server/internal/infra/database"
	"github.com/Rahmannugar/consumel-server/internal/infra/ratelimit"
	"github.com/Rahmannugar/consumel-server/internal/infra/telemetry"
	organizationrepositories "github.com/Rahmannugar/consumel-server/internal/organizations/repositories"
	userrepositories "github.com/Rahmannugar/consumel-server/internal/users/repositories"
	userservices "github.com/Rahmannugar/consumel-server/internal/users/services"
	"github.com/resend/resend-go/v2"
)

const (
	databaseTimeout       = 10 * time.Second
	authMigrationTimeout  = 30 * time.Second
	resendTimeout         = 10 * time.Second
	googleTimeout         = 10 * time.Second
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
	// Telemetry is initialized first and shut down last so dependency close and
	// HTTP shutdown failures can still be correlated and exported.
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
	databaseContext, cancelDatabase := context.WithTimeout(context.Background(), databaseTimeout)
	databasePool, err := database.Open(databaseContext, cfg.Database.ConnectionString())
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
	passwordResetAttemptGuard := infraauthentication.NewPasswordResetAttemptGuard(distributedLimiter)

	resendClient := resend.NewCustomClient(telemetry.NewHTTPClient(resendTimeout), cfg.Resend.APIKey)
	authenticationEmailSender, err := clients.NewResendAuthenticationEmailSender(
		resendClient.Emails,
		cfg.Resend.NoReplyFrom,
	)
	if err != nil {
		return fmt.Errorf("configure authentication email sender: %w", err)
	}

	auth, err := authlier.New(authlier.Config{
		AppName:         "Consumel",
		BaseURL:         cfg.Auth.BaseURL,
		BasePath:        "/auth",
		AccountBasePath: "/account",
		Database:        authlierDatabase,
		TrustedOrigins:  cfg.Auth.TrustedOrigins,
		TrustedProxies:  cfg.Auth.TrustedProxies,
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
			Sender:                      authenticationEmailSender,
			SendOnSignUp:                true,
			AutoSignInAfterVerification: true,
			AttemptGuard:                otpAttemptGuard,
		},
		PasswordReset: authlier.PasswordResetConfig{
			Enabled:      true,
			ResetURL:     cfg.Auth.PasswordResetURL(),
			Sender:       authenticationEmailSender,
			AttemptGuard: passwordResetAttemptGuard,
		},
		Session: authlier.SessionConfig{
			Mode:     authlier.SessionModeCookie,
			Lifetime: sessionLifetime,
			Cache:    sessionCache,
			CacheTTL: sessionCacheTTL,
			Cookie: authlier.CookieConfig{
				Name:     cfg.SessionCookieName(),
				Path:     "/",
				SameSite: http.SameSiteLaxMode,
			},
		},
		Google: authlier.GoogleConfig{
			Enabled:            cfg.Auth.GoogleEnabled(),
			ClientID:           cfg.Auth.GoogleClientID,
			ClientSecret:       cfg.Auth.GoogleClientSecret,
			SuccessRedirectURL: cfg.Auth.GoogleSuccessURL(),
			HTTPClient:         telemetry.NewHTTPClient(googleTimeout),
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
	router, err := newRouter(
		cfg,
		telemetryRuntime,
		databasePool,
		auth.Handler(),
		tenantService,
		distributedLimiter,
		logger,
	)
	if err != nil {
		return err
	}
	return serveHTTP(cfg.HTTP.Address(), router, logger)
}
