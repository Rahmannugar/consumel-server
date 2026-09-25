package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Rahmannugar/consumel-server/internal/common/httpresponse"
	"github.com/Rahmannugar/consumel-server/internal/infra/ratelimit"
	"github.com/gin-gonic/gin"
)

var (
	authReadPerIPLimit = ratelimit.Rule{
		ImmediateRequests:            60,
		RestoreImmediateRequestsOver: time.Minute,
		MaximumRequests:              300,
		MaximumWindow:                5 * time.Minute,
	}
	authWritePerIPLimit = ratelimit.Rule{
		ImmediateRequests:            20,
		RestoreImmediateRequestsOver: time.Minute,
		MaximumRequests:              100,
		MaximumWindow:                5 * time.Minute,
	}
)

func RateLimitRequests(limiter *ratelimit.RedisLimiter, logger *slog.Logger) gin.HandlerFunc {
	return func(context *gin.Context) {
		rule := authWritePerIPLimit
		operation := "write"
		if context.Request.Method == http.MethodGet {
			rule = authReadPerIPLimit
			operation = "read"
		}

		decision, err := limiter.Allow(
			context.Request.Context(),
			"authentication.http."+operation,
			"ip",
			requestIP(context.ClientIP()),
			rule,
		)
		if errors.Is(err, ratelimit.ErrLimitExceeded) {
			retryAfterSeconds := max(1, int((decision.RetryAfter+time.Second-1)/time.Second))
			context.Header("Retry-After", strconv.Itoa(retryAfterSeconds))
			_ = httpresponse.WriteError(
				context.Writer,
				http.StatusTooManyRequests,
				"rate_limit_exceeded",
				"Too many authentication requests. Wait before trying again.",
			)
			context.Abort()
			return
		}
		if err != nil {
			// This coarse traffic limit fails open so account and session reads can
			// fall back to PostgreSQL. Credential-specific attempt guards fail closed.
			logger.WarnContext(context.Request.Context(), "authentication request rate limit unavailable",
				"event", "authentication.rate_limit.unavailable",
				"operation", operation,
				"method", strings.ToUpper(context.Request.Method),
				"outcome", "allowed_without_rate_limit",
				"error", err,
			)
		}

		context.Next()
	}
}

func requestIP(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	return value
}
