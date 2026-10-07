package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	authenticationhandlers "github.com/Rahmannugar/consumel-server/internal/authentication/handlers"
	authenticationservices "github.com/Rahmannugar/consumel-server/internal/authentication/services"
	"github.com/Rahmannugar/consumel-server/internal/common/httpresponse"
	"github.com/Rahmannugar/consumel-server/internal/infra/telemetry"
	projectmodels "github.com/Rahmannugar/consumel-server/internal/projects/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PortfolioService interface {
	Get(context.Context, uuid.UUID, projectmodels.PortfolioFilter, string, string) (projectmodels.Portfolio, error)
}

type portfolioHandler struct {
	resolver authenticationhandlers.TenantResolver
	service  PortfolioService
	logger   *slog.Logger
}

func registerPortfolioRoute(
	router gin.IRouter,
	resolver authenticationhandlers.TenantResolver,
	service PortfolioService,
	logger *slog.Logger,
) {
	handler := &portfolioHandler{resolver: resolver, service: service, logger: logger}
	router.GET("/v1/projects/portfolio", portfolioTelemetry(), gin.WrapF(handler.get))
}

func (handler *portfolioHandler) get(response http.ResponseWriter, request *http.Request) {
	tenant, err := handler.resolver.Resolve(request)
	if err != nil {
		if errors.Is(err, authenticationservices.ErrUnauthenticated) {
			_ = httpresponse.WriteError(response, http.StatusUnauthorized, "not_authenticated",
				"Sign in to view your project portfolio.")
			return
		}
		handler.fail(response, request, err)
		return
	}

	portfolio, err := handler.service.Get(
		request.Context(),
		tenant.User.ID,
		projectmodels.PortfolioFilter{
			Environment: projectmodels.ProjectEnvironmentName(request.URL.Query().Get("environment")),
			Interval:    projectmodels.PortfolioInterval(request.URL.Query().Get("interval")),
		},
		request.URL.Query().Get("from"),
		request.URL.Query().Get("to"),
	)
	if errors.Is(err, projectmodels.ErrPortfolioFilterInvalid) {
		_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_filter",
			"Choose a valid environment, interval, and date range. Hourly ranges may span 31 days; daily ranges may span one year.")
		return
	}
	if err != nil {
		handler.fail(response, request, err)
		return
	}

	telemetry.AddRequestLogAttributes(request.Context(),
		slog.String("environment", string(portfolio.Environment)),
		slog.Int("project_count", len(portfolio.Projects)),
		slog.Int("analytics_bucket_count", len(portfolio.Buckets)),
	)
	if err := httpresponse.WriteJSON(response, http.StatusOK, portfolioJSON(portfolio)); err != nil {
		handler.logger.ErrorContext(request.Context(), "Could not send project portfolio response",
			"event", "projects.portfolio.response.failed",
			"operation", "projects.portfolio.respond",
			"outcome", "error",
			"error", err,
		)
	}
}

func (handler *portfolioHandler) fail(response http.ResponseWriter, request *http.Request, err error) {
	handler.logger.ErrorContext(request.Context(), "Could not load project portfolio",
		"event", "projects.portfolio.failed",
		"operation", "projects.portfolio",
		"outcome", "error",
		"error", err,
	)
	_ = httpresponse.WriteError(response, http.StatusInternalServerError, "project_portfolio_failed",
		"Consumel could not load the project portfolio. Try again shortly.")
}
