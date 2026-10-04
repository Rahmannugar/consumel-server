package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	authenticationhandlers "github.com/Rahmannugar/consumel-server/internal/authentication/handlers"
	"github.com/Rahmannugar/consumel-server/internal/common/httpresponse"
	consumptionmodels "github.com/Rahmannugar/consumel-server/internal/core/consumption/models"
	consumptionservices "github.com/Rahmannugar/consumel-server/internal/core/consumption/services"
	"github.com/Rahmannugar/consumel-server/internal/infra/telemetry"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type OperationService interface {
	List(context.Context, uuid.UUID, *consumptionmodels.OperationListCursor, int, consumptionmodels.OperationListFilter, string, string) ([]consumptionmodels.Operation, *consumptionmodels.OperationListCursor, error)
	Next(context.Context, uuid.UUID, string, time.Duration) (consumptionmodels.Operation, string, bool, error)
}

type OperationHandler struct {
	resolver   authenticationhandlers.TenantResolver
	authorizer ProjectEnvironmentAuthorizer
	service    OperationService
	logger     *slog.Logger
	stream     *telemetry.SSEObserver
}

func RegisterOperationRoutes(
	router gin.IRouter,
	resolver authenticationhandlers.TenantResolver,
	authorizer ProjectEnvironmentAuthorizer,
	service OperationService,
	stream *telemetry.SSEObserver,
	logger *slog.Logger,
) {
	handler := &OperationHandler{
		resolver: resolver, authorizer: authorizer, service: service, stream: stream, logger: logger,
	}
	router.GET(
		"/v1/projects/:projectID/environments/:environment/events",
		balanceDashboardPathParameters(),
		operationTelemetry(),
		gin.WrapF(handler.List),
	)
	router.GET(
		"/v1/projects/:projectID/environments/:environment/events/stream",
		balanceDashboardPathParameters(),
		gin.WrapF(handler.Stream),
	)
}

const operationStreamBlock = 15 * time.Second

func (handler *OperationHandler) Stream(response http.ResponseWriter, request *http.Request) {
	environmentID, ok := resolveDashboardEnvironment(
		response, request, handler.resolver, handler.authorizer,
		"Activate Live before streaming its events.",
		func(err error) { handler.fail(response, request, err) },
	)
	if !ok {
		handler.stream.Rejected(request.Context())
		return
	}
	after := strings.TrimSpace(request.Header.Get("Last-Event-ID"))
	if after == "" {
		after = strings.TrimSpace(request.URL.Query().Get("cursor"))
	}
	if !validStreamCursor(after) {
		handler.stream.Rejected(request.Context())
		_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_cursor", "Reconnect with the last event cursor returned by Consumel.")
		return
	}
	response.Header().Set("Content-Type", "text/event-stream")
	response.Header().Set("Cache-Control", "no-cache, no-transform")
	response.Header().Set("Connection", "keep-alive")
	response.Header().Set("X-Accel-Buffering", "no")
	response.WriteHeader(http.StatusOK)
	controller := http.NewResponseController(response)
	if err := controller.Flush(); err != nil {
		return
	}
	disconnected := handler.stream.Connected(request.Context())
	defer disconnected()

	for {
		operation, cursor, found, err := handler.service.Next(
			request.Context(), environmentID, after, operationStreamBlock,
		)
		after = cursor
		if err != nil {
			// A canceled request is the normal end of an SSE connection, not a
			// delivery failure that should page operators or inflate error metrics.
			if request.Context().Err() != nil || errors.Is(err, context.Canceled) {
				return
			}
			handler.stream.DeliveryFailed(request.Context())
			handler.logger.ErrorContext(request.Context(), "Could not deliver usage event stream",
				"event", "operations.stream.delivery_failed", "operation", "operations.stream",
				"outcome", "error", "error", err,
			)
			return
		}
		if !found {
			if _, err := response.Write([]byte(": heartbeat\n\n")); err != nil {
				return
			}
		} else {
			payload, err := json.Marshal(operationJSON(operation))
			if err != nil {
				handler.logger.ErrorContext(request.Context(), "Could not encode usage stream event",
					"event", "operations.stream.encoding_failed", "operation", "operations.stream",
					"outcome", "error", "error", err,
				)
				return
			}
			if _, err := fmt.Fprintf(response, "id: %s\nevent: usage.operation\ndata: %s\n\n", after, payload); err != nil {
				return
			}
		}
		if err := controller.Flush(); err != nil {
			return
		}
	}
}

func validStreamCursor(value string) bool {
	if value == "" {
		return true
	}
	parts := strings.Split(value, "-")
	if len(parts) != 2 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		if _, err := strconv.ParseUint(part, 10, 64); err != nil {
			return false
		}
	}
	return true
}

func (handler *OperationHandler) List(response http.ResponseWriter, request *http.Request) {
	environmentID, ok := resolveDashboardEnvironment(
		response, request, handler.resolver, handler.authorizer,
		"Activate Live before viewing its events.",
		func(err error) { handler.fail(response, request, err) },
	)
	if !ok {
		return
	}
	cursor, err := consumptionmodels.DecodeOperationCursor(request.URL.Query().Get("cursor"))
	if err != nil {
		_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_cursor", "Use the next cursor returned by Consumel.")
		return
	}
	limit, err := operationListLimit(request.URL.Query().Get("limit"))
	if err != nil {
		_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_request", "Limit must be a number between 1 and 100.")
		return
	}
	operations, next, err := handler.service.List(
		request.Context(), environmentID, cursor, limit,
		consumptionmodels.OperationListFilter{
			Status:     consumptionmodels.OperationStatus(request.URL.Query().Get("status")),
			CustomerID: request.URL.Query().Get("customerId"),
			MeterKey:   request.URL.Query().Get("meterKey"),
		},
		request.URL.Query().Get("from"),
		request.URL.Query().Get("to"),
	)
	if errors.Is(err, consumptionmodels.ErrOperationFilterInvalid) {
		_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_filter", "Choose valid customer, meter, outcome, and date filters with a range no longer than one year.")
		return
	}
	if err != nil {
		handler.fail(response, request, err)
		return
	}
	telemetry.AddRequestLogAttributes(request.Context(), slog.Int("operation_count", len(operations)))
	if err := httpresponse.WriteJSON(response, http.StatusOK, operationListJSON(operations, next)); err != nil {
		handler.logger.ErrorContext(request.Context(), "Could not send operation list response",
			"event", "operations.response.failed", "operation", "operations.list",
			"outcome", "error", "error", err,
		)
	}
}

func (handler *OperationHandler) fail(response http.ResponseWriter, request *http.Request, err error) {
	handler.logger.ErrorContext(request.Context(), "Could not list consumption operations",
		"event", "operations.list.failed", "operation", "operations.list",
		"outcome", "error", "error", err,
	)
	_ = httpresponse.WriteError(response, http.StatusInternalServerError, "operation_list_failed", "Consumel could not load events. Try again shortly.")
}

func operationListLimit(value string) (int, error) {
	if strings.TrimSpace(value) == "" {
		return consumptionservices.DefaultOperationPageSize, nil
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit < 1 || limit > consumptionservices.MaximumOperationPageSize {
		return 0, errors.New("invalid operation page size")
	}
	return limit, nil
}
