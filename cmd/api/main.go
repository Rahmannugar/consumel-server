package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/Rahmannugar/consumel-server/internal/config"
	consumptionrepositories "github.com/Rahmannugar/consumel-server/internal/core/consumption/repositories"
	consumptionservices "github.com/Rahmannugar/consumel-server/internal/core/consumption/services"
	customerrepositories "github.com/Rahmannugar/consumel-server/internal/customers/repositories"
	customerservices "github.com/Rahmannugar/consumel-server/internal/customers/services"
	"github.com/Rahmannugar/consumel-server/internal/infra/cache"
	"github.com/Rahmannugar/consumel-server/internal/infra/database"
	"github.com/Rahmannugar/consumel-server/internal/infra/emaildelivery"
	"github.com/Rahmannugar/consumel-server/internal/infra/events"
	"github.com/Rahmannugar/consumel-server/internal/infra/telemetry"
	meterrepositories "github.com/Rahmannugar/consumel-server/internal/meters/repositories"
	meterservices "github.com/Rahmannugar/consumel-server/internal/meters/services"
	onboardingrepositories "github.com/Rahmannugar/consumel-server/internal/onboarding/repositories"
	onboardingservices "github.com/Rahmannugar/consumel-server/internal/onboarding/services"
	projectrepositories "github.com/Rahmannugar/consumel-server/internal/projects/repositories"
	projectservices "github.com/Rahmannugar/consumel-server/internal/projects/services"
)

const (
	databaseTimeout = 10 * time.Second
	// apiPoolDefault caps the API's client-side database pool.
	apiPoolDefault int32 = 20
)

// @title Consumel API
// @version 0.1.0
// @description Consumel usage-based billing infrastructure API.
// @servers.url https://api.consumel.com
// @servers.description Production
// @servers.url http://localhost:8080
// @servers.description Local development

// @SecurityDefinitions.apikey localCookieSession
// @in cookie
// @name consumel_session

// @SecurityDefinitions.apikey productionCookieSession
// @in cookie
// @name __Host-consumel_session

// @SecurityDefinitions.bearerauth projectAPIKey
// @bearerformat cm_test_… or cm_live_…
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
	telemetryRuntime, err := telemetry.New(context.Background(), "consumel-api", string(cfg.Environment))
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
	databasePool, err := database.Open(databaseContext, cfg.Database.ConnectionString(), cfg.Database.APIPoolOr(apiPoolDefault))
	cancelDatabase()
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer databasePool.Close()

	redisClient, err := cache.Open(cfg.Redis.URL)
	if err != nil {
		return fmt.Errorf("connect Redis: %w", err)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			logger.Error("close Redis client", "error", err)
		}
	}()

	emailQueue, err := emaildelivery.NewQueue(databasePool, cfg.Auth.OTPHMACSecret)
	if err != nil {
		return fmt.Errorf("configure email delivery queue: %w", err)
	}
	authentication, err := newAuthenticationComponents(
		cfg,
		databasePool,
		redisClient,
		emailQueue,
		logger,
	)
	if err != nil {
		return err
	}
	onboardingService := onboardingservices.New(
		onboardingrepositories.New(databasePool, emailQueue),
		cfg.Auth.ClientBaseURL,
	)
	projectService := projectservices.NewProjectManagementService(
		projectrepositories.NewProjectRepository(databasePool),
	)
	customerService := customerservices.NewCustomerService(
		customerrepositories.NewCustomerRepository(databasePool),
	)
	meterService := meterservices.NewMeterService(
		meterrepositories.NewMeterRepository(databasePool),
	)
	balanceService := consumptionservices.NewBalanceService(
		consumptionrepositories.NewBalanceRepository(databasePool),
	)
	consumeService := consumptionservices.NewConsumeService(
		consumptionrepositories.NewConsumeRepository(databasePool),
	)
	operationService := consumptionservices.NewOperationService(
		consumptionrepositories.NewOperationRepository(databasePool),
		events.NewOperationSource(redisClient),
	)
	analyticsService := consumptionservices.NewAnalyticsService(
		consumptionrepositories.NewAnalyticsRepository(databasePool),
	)
	router, err := newRouter(
		cfg,
		telemetryRuntime,
		databasePool,
		authentication,
		onboardingService,
		projectService,
		customerService,
		meterService,
		balanceService,
		consumeService,
		operationService,
		analyticsService,
		logger,
	)
	if err != nil {
		return err
	}
	return serveHTTP(cfg.HTTP.Address(), router, logger)
}
