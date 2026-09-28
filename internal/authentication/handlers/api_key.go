package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	authenticationmodels "github.com/Rahmannugar/consumel-server/internal/authentication/models"
	authenticationservices "github.com/Rahmannugar/consumel-server/internal/authentication/services"
	"github.com/Rahmannugar/consumel-server/internal/common/httpresponse"
	"github.com/Rahmannugar/consumel-server/internal/infra/telemetry"
	"github.com/gin-gonic/gin"
)

type APIKeyAuthenticator interface {
	Authenticate(context.Context, string) (authenticationmodels.APIKeyContext, error)
}

type apiKeyContextKey struct{}

func RequireProjectAPIKey(
	authenticator APIKeyAuthenticator,
	logger *slog.Logger,
) gin.HandlerFunc {
	return func(ginContext *gin.Context) {
		request := ginContext.Request
		resolved, err := authenticator.Authenticate(
			request.Context(), request.Header.Get("Authorization"),
		)
		if errors.Is(err, authenticationservices.ErrInvalidAPIKey) {
			_ = httpresponse.WriteError(ginContext.Writer, http.StatusUnauthorized,
				"invalid_api_key", "Provide an active project environment API key.")
			ginContext.Abort()
			return
		}
		if err != nil {
			logger.ErrorContext(request.Context(), "Could not authenticate project API key",
				"event", "api_key.authentication.failed",
				"operation", "api_key.authenticate", "outcome", "error", "error", err,
			)
			_ = httpresponse.WriteError(ginContext.Writer, http.StatusInternalServerError,
				"authentication_failed", "Consumel could not authenticate the request. Try again shortly.")
			ginContext.Abort()
			return
		}

		telemetry.AddRequestLogAttributes(request.Context(),
			slog.String("organization_id", resolved.OrganizationID.String()),
			slog.String("project_id", resolved.ProjectID.String()),
			slog.String("project_environment_id", resolved.ProjectEnvironmentID.String()),
			slog.String("environment", resolved.Environment),
		)
		ginContext.Request = request.WithContext(
			context.WithValue(request.Context(), apiKeyContextKey{}, resolved),
		)
		ginContext.Next()
	}
}

func ProjectAPIKeyContext(ctx context.Context) (authenticationmodels.APIKeyContext, bool) {
	resolved, ok := ctx.Value(apiKeyContextKey{}).(authenticationmodels.APIKeyContext)
	return resolved, ok
}
