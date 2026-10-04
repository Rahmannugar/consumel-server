package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	authenticationhandlers "github.com/Rahmannugar/consumel-server/internal/authentication/handlers"
	"github.com/Rahmannugar/consumel-server/internal/common/httpresponse"
	consumptionmodels "github.com/Rahmannugar/consumel-server/internal/core/consumption/models"
	"github.com/Rahmannugar/consumel-server/internal/infra/telemetry"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AnalyticsService interface {
	Get(context.Context, uuid.UUID, consumptionmodels.AnalyticsFilter, string, string) (consumptionmodels.Analytics, error)
}

type AnalyticsHandler struct {
	resolver   authenticationhandlers.TenantResolver
	authorizer ProjectEnvironmentAuthorizer
	service    AnalyticsService
	logger     *slog.Logger
}

func RegisterAnalyticsRoutes(
	router gin.IRouter,
	apiKeyAuthenticator authenticationhandlers.APIKeyAuthenticator,
	resolver authenticationhandlers.TenantResolver,
	authorizer ProjectEnvironmentAuthorizer,
	service AnalyticsService,
	logger *slog.Logger,
) {
	handler := &AnalyticsHandler{resolver: resolver, authorizer: authorizer, service: service, logger: logger}
	public := router.Group("/v1/analytics", authenticationhandlers.RequireProjectAPIKey(apiKeyAuthenticator, logger))
	public.GET("", analyticsTelemetry("analytics.get"), gin.WrapF(handler.Get))
	dashboard := router.Group(
		"/v1/projects/:projectID/environments/:environment/analytics",
		balanceDashboardPathParameters(),
	)
	dashboard.GET("", analyticsTelemetry("analytics.dashboard.get"), gin.WrapF(handler.Get))
}

func (handler *AnalyticsHandler) Get(response http.ResponseWriter, request *http.Request) {
	environmentID, ok := handler.environmentID(response, request)
	if !ok {
		return
	}
	analytics, err := handler.service.Get(
		request.Context(),
		environmentID,
		consumptionmodels.AnalyticsFilter{
			Interval:   consumptionmodels.AnalyticsInterval(request.URL.Query().Get("interval")),
			CustomerID: request.URL.Query().Get("customerId"),
			MeterKey:   request.URL.Query().Get("meterKey"),
		},
		request.URL.Query().Get("from"),
		request.URL.Query().Get("to"),
	)
	if errors.Is(err, consumptionmodels.ErrAnalyticsFilterInvalid) {
		_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_filter", "Choose valid customer, meter, interval, and date filters. Hourly ranges may span 31 days; daily ranges may span one year.")
		return
	}
	if err != nil {
		handler.fail(response, request, err)
		return
	}
	telemetry.AddRequestLogAttributes(request.Context(), slog.Int("analytics_bucket_count", len(analytics.Buckets)))
	if err := httpresponse.WriteJSON(response, http.StatusOK, analyticsJSON(analytics)); err != nil {
		handler.logger.ErrorContext(request.Context(), "Could not send analytics response",
			"event", "analytics.response.failed", "operation", "analytics.get",
			"outcome", "error", "error", err,
		)
	}
}

func (handler *AnalyticsHandler) environmentID(response http.ResponseWriter, request *http.Request) (uuid.UUID, bool) {
	if resolved, ok := authenticationhandlers.ProjectAPIKeyContext(request.Context()); ok {
		return resolved.ProjectEnvironmentID, true
	}
	return resolveDashboardEnvironment(
		response, request, handler.resolver, handler.authorizer,
		"Activate Live before viewing its analytics.",
		func(err error) { handler.fail(response, request, err) },
	)
}

func (handler *AnalyticsHandler) fail(response http.ResponseWriter, request *http.Request, err error) {
	handler.logger.ErrorContext(request.Context(), "Could not load consumption analytics",
		"event", "analytics.get.failed", "operation", "analytics.get",
		"outcome", "error", "error", err,
	)
	_ = httpresponse.WriteError(response, http.StatusInternalServerError, "analytics_failed", "Consumel could not load analytics. Try again shortly.")
}
