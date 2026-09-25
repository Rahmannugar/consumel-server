package handlers

import (
	"log/slog"
	"net/http"

	"github.com/Rahmannugar/consumel-server/internal/infra/ratelimit"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes keeps authentication transport policy with the domain while
// Authlier remains the reusable net/http implementation behind those routes.
func RegisterRoutes(
	router gin.IRouter,
	authlierHandler http.Handler,
	resolver TenantResolver,
	limiter *ratelimit.RedisLimiter,
	logger *slog.Logger,
) {
	middleware := []gin.HandlerFunc{RequestTelemetry(), RateLimitRequests(limiter, logger)}

	authentication := router.Group("/auth", middleware...)
	authentication.Any("/*path", gin.WrapH(authlierHandler))

	accountHandler := NewAccountHandler(resolver, logger)
	account := router.Group("/account", middleware...)
	account.GET("", gin.WrapF(accountHandler.Get))
	account.Any("/*path", gin.WrapH(authlierHandler))
}
