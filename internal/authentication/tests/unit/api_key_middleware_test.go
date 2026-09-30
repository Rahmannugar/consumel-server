package unit_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authenticationhandlers "github.com/Rahmannugar/consumel-server/internal/authentication/handlers"
	authenticationmodels "github.com/Rahmannugar/consumel-server/internal/authentication/models"
	authenticationservices "github.com/Rahmannugar/consumel-server/internal/authentication/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type middlewareAuthenticator struct {
	resolved authenticationmodels.APIKeyContext
	err      error
}

func (authenticator middlewareAuthenticator) Authenticate(
	_ context.Context,
	_ string,
) (authenticationmodels.APIKeyContext, error) {
	return authenticator.resolved, authenticator.err
}

func TestAPIKeyMiddlewareEstablishesProjectEnvironmentContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	environmentID := uuid.MustParse("0199a417-1ae1-7b67-ad5b-809be2f9ca0a")
	router := gin.New()
	router.GET("/v1/customers", authenticationhandlers.RequireProjectAPIKey(
		middlewareAuthenticator{resolved: authenticationmodels.APIKeyContext{
			ProjectEnvironmentID: environmentID,
		}},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	), func(context *gin.Context) {
		resolved, ok := authenticationhandlers.ProjectAPIKeyContext(context.Request.Context())
		if !ok || resolved.ProjectEnvironmentID != environmentID {
			context.Status(http.StatusInternalServerError)
			return
		}
		context.Status(http.StatusNoContent)
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/customers", nil)
	request.Header.Set("Authorization", "Bearer active-key")
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
}

func TestAPIKeyMiddlewareReturnsBoundedUnauthorizedResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/v1/customers", authenticationhandlers.RequireProjectAPIKey(
		middlewareAuthenticator{err: authenticationservices.ErrInvalidAPIKey},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	), func(context *gin.Context) {
		context.Status(http.StatusNoContent)
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/customers", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if !strings.Contains(response.Body.String(), `"code":"invalid_api_key"`) {
		t.Fatalf("body = %s, want bounded invalid_api_key response", response.Body.String())
	}
}
