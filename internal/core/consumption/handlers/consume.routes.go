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

type ConsumeService interface {
	Consume(context.Context, uuid.UUID, uuid.UUID, string, consumptionmodels.ConsumeRequest) (consumptionmodels.UsageEvent, bool, error)
}

type ConsumeHandler struct {
	service ConsumeService
	logger  *slog.Logger
}

func RegisterConsumeRoutes(
	router gin.IRouter,
	authenticator authenticationhandlers.APIKeyAuthenticator,
	service ConsumeService,
	logger *slog.Logger,
) {
	handler := &ConsumeHandler{service: service, logger: logger}
	router.POST(
		"/v1/consume",
		authenticationhandlers.RequireProjectAPIKey(authenticator, logger),
		consumeTelemetry(),
		gin.WrapF(handler.Consume),
	)
}

func (handler *ConsumeHandler) Consume(response http.ResponseWriter, request *http.Request) {
	authenticated, ok := authenticationhandlers.ProjectAPIKeyContext(request.Context())
	if !ok {
		_ = httpresponse.WriteError(response, http.StatusUnauthorized, "invalid_api_key", "Use an active project API key.")
		return
	}
	var input consumptionmodels.ConsumeRequest
	if !decodeBalanceRequest(response, request, &input) {
		return
	}
	result, replayed, err := handler.service.Consume(
		request.Context(),
		authenticated.ProjectEnvironmentID,
		authenticated.APIKeyID,
		request.Header.Get(idempotencyKeyHeader),
		input,
	)
	if replayed {
		response.Header().Set(idempotencyReplayHeader, "true")
	}
	if err != nil {
		handler.writeConsumeError(response, request, err)
		return
	}
	telemetry.AddRequestLogAttributes(request.Context(),
		slog.String("usage_event_id", result.ID.String()),
		slog.String("meter_type", string(result.MeterType)),
		slog.Bool("billable", result.Billable),
		slog.Bool("idempotency_replayed", replayed),
	)
	if err := httpresponse.WriteJSON(response, http.StatusOK, usageEventJSON(result)); err != nil {
		handler.logger.ErrorContext(request.Context(), "Could not send consume response",
			"event", "consumption.response.failed", "operation", "consumption.respond",
			"outcome", "error", "error", err,
		)
	}
}

func (handler *ConsumeHandler) writeConsumeError(
	response http.ResponseWriter,
	request *http.Request,
	err error,
) {
	switch {
	case errors.Is(err, consumptionmodels.ErrConsumeCustomerInvalid),
		errors.Is(err, consumptionmodels.ErrConsumeMeterInvalid),
		errors.Is(err, consumptionmodels.ErrConsumeQuantityInvalid),
		errors.Is(err, consumptionmodels.ErrIdempotencyKeyInvalid):
		_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_consume", "Check the customer ID, meter key, quantity, and idempotency key.")
	case errors.Is(err, consumptionmodels.ErrConsumeMeterNotFound):
		_ = httpresponse.WriteError(response, http.StatusNotFound, "meter_not_found", "The active meter does not exist in this environment.")
	case errors.Is(err, consumptionmodels.ErrInsufficientBalance):
		_ = httpresponse.WriteError(response, http.StatusConflict, "insufficient_balance", "The customer does not have enough available balance for this prepaid usage.")
	case errors.Is(err, consumptionmodels.ErrIdempotencyKeyConflict):
		_ = httpresponse.WriteError(response, http.StatusConflict, "idempotency_key_conflict", "Use a new idempotency key for a different consume request.")
	default:
		handler.logger.ErrorContext(request.Context(), "Could not complete consume request",
			"event", "consumption.request.failed", "operation", "consumption.request",
			"outcome", "error", "error", err,
		)
		_ = httpresponse.WriteError(response, http.StatusInternalServerError, "consume_failed", "Consumel could not process this usage. Try again shortly.")
	}
}
