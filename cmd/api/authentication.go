package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/Rahmannugar/authlier"
	"github.com/Rahmannugar/authlier/emailverification"
	"github.com/Rahmannugar/authlier/googleoauth"
	"github.com/Rahmannugar/authlier/sessiontoken"
	authlierpostgres "github.com/Rahmannugar/authlier/storage/postgres"
	authlierredis "github.com/Rahmannugar/authlier/storage/redis"
	consumelauthentication "github.com/Rahmannugar/consumel-server/internal/authentication"
	authenticationrepositories "github.com/Rahmannugar/consumel-server/internal/authentication/repositories"
	authenticationservices "github.com/Rahmannugar/consumel-server/internal/authentication/services"
	"github.com/Rahmannugar/consumel-server/internal/config"
	infraauthentication "github.com/Rahmannugar/consumel-server/internal/infra/authentication"
	"github.com/Rahmannugar/consumel-server/internal/infra/emaildelivery"
	"github.com/Rahmannugar/consumel-server/internal/infra/ratelimit"
	"github.com/Rahmannugar/consumel-server/internal/infra/telemetry"
	userrepositories "github.com/Rahmannugar/consumel-server/internal/users/repositories"
	userservices "github.com/Rahmannugar/consumel-server/internal/users/services"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	authMigrationTimeout    = 30 * time.Second
	googleTimeout           = 10 * time.Second
	sessionIdleLifetime     = 7 * 24 * time.Hour
	sessionExtensionAfter   = 24 * time.Hour
	sessionAbsoluteLifetime = 30 * 24 * time.Hour
	sessionCacheTTL         = time.Hour
	maximumActiveSessions   = 3
)

type authenticationComponents struct {
	handler             http.Handler
	tenantResolver      *authenticationservices.AuthenticatedTenantService
	apiKeyAuthenticator *authenticationservices.APIKeyAuthenticator
	limiter             *ratelimit.RedisLimiter
}

func newAuthenticationComponents(
	cfg config.Config,
	databasePool *pgxpool.Pool,
	redisClient *redis.Client,
	emailQueue *emaildelivery.Queue,
	logger *slog.Logger,
) (authenticationComponents, error) {
	authlierPostgres, err := authlierpostgres.New(databasePool, authlierpostgres.Config{})
	if err != nil {
		return authenticationComponents{}, fmt.Errorf("configure Authlier PostgreSQL storage: %w", err)
	}
	authMigrationContext, cancelAuthMigration := context.WithTimeout(
		context.Background(),
		authMigrationTimeout,
	)
	err = authlierPostgres.Migrate(authMigrationContext)
	cancelAuthMigration()
	if err != nil {
		return authenticationComponents{}, fmt.Errorf("migrate Authlier PostgreSQL storage: %w", err)
	}

	redisSessionCache, err := authlierredis.NewSessionCache(redisClient, "consumel:auth")
	if err != nil {
		return authenticationComponents{}, fmt.Errorf("configure Authlier session cache: %w", err)
	}
	sessionCache, err := infraauthentication.NewJitteredSessionCache(redisSessionCache)
	if err != nil {
		return authenticationComponents{}, fmt.Errorf("configure jittered session cache: %w", err)
	}
	emailVerificationStore, err := infraauthentication.NewRedisEmailVerificationStore(
		databasePool,
		redisClient,
		cfg.Auth.OTPHMACSecret,
		"consumel:auth:email-verification",
	)
	if err != nil {
		return authenticationComponents{}, fmt.Errorf("configure Redis email verification storage: %w", err)
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
		return authenticationComponents{}, fmt.Errorf("configure Consumel Authlier storage policy: %w", err)
	}
	distributedLimiter, err := ratelimit.NewRedisLimiter(
		redisClient,
		cfg.Auth.OTPHMACSecret,
		"consumel:rate-limit",
	)
	if err != nil {
		return authenticationComponents{}, fmt.Errorf("configure distributed authentication rate limiter: %w", err)
	}

	googleProvider, err := newGoogleProvider(cfg, logger)
	if err != nil {
		return authenticationComponents{}, err
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
			AttemptGuard:             infraauthentication.NewPasswordAttemptGuard(distributedLimiter),
		},
		EmailVerification: authlier.EmailVerificationConfig{
			Enabled:                     true,
			Delivery:                    emailverification.DeliveryMethodOTP,
			OTPSecret:                   cfg.Auth.OTPHMACSecret,
			Sender:                      emailQueue,
			SendOnSignUp:                true,
			SendOnSignIn:                true,
			AutoSignInAfterVerification: true,
			AttemptGuard:                infraauthentication.NewOTPAttemptGuard(distributedLimiter),
		},
		PasswordReset: authlier.PasswordResetConfig{
			Enabled:      true,
			ResetURL:     cfg.Auth.PasswordResetURL(),
			Sender:       emailQueue,
			AttemptGuard: infraauthentication.NewPasswordResetAttemptGuard(distributedLimiter),
		},
		Session: authlier.SessionConfig{
			Mode:     authlier.SessionModeCookie,
			Lifetime: sessionIdleLifetime,
			Cache:    sessionCache,
			CacheTTL: sessionCacheTTL,
			Extension: &sessiontoken.ExtensionConfig{
				ExtendAfter:      sessionExtensionAfter,
				AbsoluteLifetime: sessionAbsoluteLifetime,
			},
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
			Provider:           googleProvider,
		},
	})
	if err != nil {
		return authenticationComponents{}, fmt.Errorf("configure Authlier: %w", err)
	}

	userService := userservices.NewUserService(userrepositories.NewUserRepository(databasePool))
	return authenticationComponents{
		handler: auth.Handler(),
		tenantResolver: authenticationservices.NewAuthenticatedTenantService(
			infraauthentication.NewAuthlierSessionResolver(auth),
			userService,
			authenticationrepositories.NewAccountContextRepository(databasePool),
		),
		apiKeyAuthenticator: authenticationservices.NewAPIKeyAuthenticator(
			authenticationrepositories.NewAPIKeyContextRepository(databasePool),
		),
		limiter: distributedLimiter,
	}, nil
}

func newGoogleProvider(cfg config.Config, logger *slog.Logger) (googleoauth.Provider, error) {
	if !cfg.Auth.GoogleEnabled() {
		return nil, nil
	}
	provider, err := googleoauth.NewGoogleProvider(googleoauth.GoogleProviderConfig{
		ClientID:     cfg.Auth.GoogleClientID,
		ClientSecret: cfg.Auth.GoogleClientSecret,
		HTTPClient:   telemetry.NewHTTPClient(googleTimeout),
	})
	if err != nil {
		return nil, fmt.Errorf("configure Google identity provider: %w", err)
	}
	return infraauthentication.NewObservedGoogleProvider(provider, logger), nil
}
