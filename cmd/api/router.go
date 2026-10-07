package main

import (
	"fmt"
	"log/slog"

	authenticationhandlers "github.com/Rahmannugar/consumel-server/internal/authentication/handlers"
	"github.com/Rahmannugar/consumel-server/internal/config"
	consumptionhandlers "github.com/Rahmannugar/consumel-server/internal/core/consumption/handlers"
	customerhandlers "github.com/Rahmannugar/consumel-server/internal/customers/handlers"
	"github.com/Rahmannugar/consumel-server/internal/health"
	"github.com/Rahmannugar/consumel-server/internal/infra/cors"
	"github.com/Rahmannugar/consumel-server/internal/infra/telemetry"
	meterhandlers "github.com/Rahmannugar/consumel-server/internal/meters/handlers"
	onboardinghandlers "github.com/Rahmannugar/consumel-server/internal/onboarding/handlers"
	"github.com/Rahmannugar/consumel-server/internal/openapi"
	projecthandlers "github.com/Rahmannugar/consumel-server/internal/projects/handlers"
	"github.com/gin-gonic/gin"
)

func newRouter(
	cfg config.Config,
	runtime *telemetry.Runtime,
	database health.Database,
	authentication authenticationComponents,
	onboardingService onboardinghandlers.OnboardingService,
	projectService projecthandlers.ProjectService,
	portfolioService projecthandlers.PortfolioService,
	customerService customerhandlers.CustomerService,
	meterService meterhandlers.MeterService,
	balanceService consumptionhandlers.BalanceService,
	consumeService consumptionhandlers.ConsumeService,
	operationService consumptionhandlers.OperationService,
	analyticsService consumptionhandlers.AnalyticsService,
	logger *slog.Logger,
) (*gin.Engine, error) {
	if cfg.Environment != config.EnvironmentDevelopment {
		gin.SetMode(gin.ReleaseMode)
	}

	requestTelemetry, err := runtime.HTTPMiddleware()
	if err != nil {
		return nil, fmt.Errorf("configure HTTP telemetry: %w", err)
	}
	router := gin.New()
	allowedOrigins := append([]string{cfg.Auth.BaseURL}, cfg.Auth.TrustedOrigins...)
	router.Use(cors.Middleware(allowedOrigins), requestTelemetry, gin.Recovery())
	if err := router.SetTrustedProxies(cfg.Auth.TrustedProxies); err != nil {
		return nil, fmt.Errorf("configure trusted HTTP proxies: %w", err)
	}

	health.RegisterRoutes(router, database)
	if err := openapi.RegisterRoutes(router); err != nil {
		return nil, fmt.Errorf("register OpenAPI routes: %w", err)
	}
	authenticationhandlers.RegisterRoutes(
		router,
		authentication.handler,
		authentication.tenantResolver,
		authentication.limiter,
		logger,
	)
	onboardinghandlers.RegisterRoutes(router, authentication.tenantResolver, onboardingService, logger)
	projecthandlers.RegisterRoutes(router, authentication.tenantResolver, projectService, portfolioService, logger)
	customerhandlers.RegisterRoutes(
		router,
		authentication.apiKeyAuthenticator,
		authentication.tenantResolver,
		projectService,
		customerService,
		logger,
	)
	meterhandlers.RegisterRoutes(
		router,
		authentication.apiKeyAuthenticator,
		authentication.tenantResolver,
		projectService,
		meterService,
		logger,
	)
	consumptionhandlers.RegisterBalanceRoutes(
		router,
		authentication.apiKeyAuthenticator,
		authentication.tenantResolver,
		projectService,
		balanceService,
		logger,
	)
	consumptionhandlers.RegisterConsumeRoutes(
		router,
		authentication.apiKeyAuthenticator,
		consumeService,
		logger,
	)
	operationStreamObserver, err := runtime.NewSSEObserver("operations.stream")
	if err != nil {
		return nil, fmt.Errorf("configure operation stream telemetry: %w", err)
	}
	consumptionhandlers.RegisterOperationRoutes(
		router,
		authentication.tenantResolver,
		projectService,
		operationService,
		operationStreamObserver,
		logger,
	)
	consumptionhandlers.RegisterAnalyticsRoutes(
		router,
		authentication.apiKeyAuthenticator,
		authentication.tenantResolver,
		projectService,
		analyticsService,
		logger,
	)
	return router, nil
}
