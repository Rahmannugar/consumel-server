package unit_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Rahmannugar/consumel-server/internal/infra/cors"
	"github.com/gin-gonic/gin"
)

func TestPreflightAllowsDashboardBalanceMutations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(cors.Middleware([]string{"http://localhost:3000"}))
	router.OPTIONS("/v1/projects/:projectID/environments/:environment/balances", func(context *gin.Context) {
		context.Status(http.StatusNoContent)
	})

	request := httptest.NewRequestWithContext(t.Context(), http.MethodOptions, "/v1/projects/project/environments/sandbox/balances", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	request.Header.Set("Access-Control-Request-Headers", "content-type,idempotency-key")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if got := response.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type, Idempotency-Key" {
		t.Fatalf("allowed headers = %q", got)
	}
	if got := response.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST, PUT, DELETE, OPTIONS" {
		t.Fatalf("allowed methods = %q", got)
	}
	if got := response.Header().Get("Access-Control-Expose-Headers"); got != "Idempotency-Replayed" {
		t.Fatalf("exposed headers = %q", got)
	}
}
