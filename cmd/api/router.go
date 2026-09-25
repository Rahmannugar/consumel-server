package main

import (
	"fmt"
	"log/slog"
	"net/http"

	authenticationhandlers "github.com/Rahmannugar/consumel-server/internal/authentication/handlers"
	"github.com/Rahmannugar/consumel-server/internal/config"
	"github.com/Rahmannugar/consumel-server/internal/health"
	"github.com/Rahmannugar/consumel-server/internal/infra/cors"
	"github.com/Rahmannugar/consumel-server/internal/infra/ratelimit"
	"github.com/Rahmannugar/consumel-server/internal/infra/telemetry"
	"github.com/Rahmannugar/consumel-server/internal/openapi"
	"github.com/gin-gonic/gin"
)

func newRouter(
	cfg config.Config,
	runtime *telemetry.Runtime,
	database health.Database,
	authlierHandler http.Handler,
	tenantResolver authenticationhandlers.TenantResolver,
	limiter *ratelimit.RedisLimiter,
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
		authlierHandler,
		tenantResolver,
		limiter,
		logger,
	)
	return router, nil
}
