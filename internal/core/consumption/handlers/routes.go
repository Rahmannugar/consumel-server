package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	authenticationhandlers "github.com/Rahmannugar/consumel-server/internal/authentication/handlers"
	"github.com/Rahmannugar/consumel-server/internal/common/httpresponse"
	consumptionmodels "github.com/Rahmannugar/consumel-server/internal/core/consumption/models"
	consumptionservices "github.com/Rahmannugar/consumel-server/internal/core/consumption/services"
	"github.com/Rahmannugar/consumel-server/internal/infra/telemetry"
	projectmodels "github.com/Rahmannugar/consumel-server/internal/projects/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	maximumBalanceRequestBytes = 8 << 10
	idempotencyKeyHeader       = "Idempotency-Key"
	idempotencyReplayHeader    = "Idempotency-Replayed"
)

type BalanceService interface {
	Add(context.Context, uuid.UUID, consumptionmodels.BalanceMutationSource, string, consumptionmodels.AddBalanceRequest) (consumptionmodels.Balance, bool, error)
	Set(context.Context, uuid.UUID, consumptionmodels.BalanceMutationSource, string, string, consumptionmodels.SetBalanceRequest) (consumptionmodels.Balance, error)
	Get(context.Context, uuid.UUID, string, string) (consumptionmodels.Balance, error)
	List(context.Context, uuid.UUID, string) ([]consumptionmodels.Balance, error)
	ListGrants(context.Context, uuid.UUID, string, string, *consumptionmodels.OperationListCursor, int) ([]consumptionmodels.EntitlementGrant, *consumptionmodels.OperationListCursor, error)
	ListActivity(context.Context, uuid.UUID, string, string, *consumptionmodels.OperationListCursor, int) ([]consumptionmodels.BalanceActivity, *consumptionmodels.OperationListCursor, error)
}

type ProjectEnvironmentAuthorizer interface {
	AccessibleEnvironment(context.Context, uuid.UUID, uuid.UUID, projectmodels.ProjectEnvironmentName) (projectmodels.ProjectEnvironment, error)
}

type Handler struct {
	resolver   authenticationhandlers.TenantResolver
	authorizer ProjectEnvironmentAuthorizer
	service    BalanceService
	logger     *slog.Logger
}

func RegisterBalanceRoutes(
	router gin.IRouter,
	apiKeyAuthenticator authenticationhandlers.APIKeyAuthenticator,
	resolver authenticationhandlers.TenantResolver,
	authorizer ProjectEnvironmentAuthorizer,
	service BalanceService,
	logger *slog.Logger,
) {
	handler := &Handler{resolver: resolver, authorizer: authorizer, service: service, logger: logger}
	publicAuth := authenticationhandlers.RequireProjectAPIKey(apiKeyAuthenticator, logger)
	publicCustomers := router.Group("/v1/customers", publicAuth)
	publicCustomers.GET("/:customerID/balances", balancePathParameters(), balanceTelemetry("balances.list", "balances.listed", "Customer balances loaded"), gin.WrapF(handler.List))
	publicCustomers.GET("/:customerID/balances/:meterKey", balancePathParameters(), balanceTelemetry("balances.get", "balances.loaded", "Balance loaded"), gin.WrapF(handler.Get))
	publicCustomers.GET("/:customerID/balances/:meterKey/grants", balancePathParameters(), balanceTelemetry("balances.grants.list", "balances.grants.listed", "Balance grants loaded"), gin.WrapF(handler.ListGrants))
	publicCustomers.GET("/:customerID/balances/:meterKey/history", balancePathParameters(), balanceTelemetry("balances.history.list", "balances.history.listed", "Balance history loaded"), gin.WrapF(handler.ListActivity))
	publicBalances := router.Group("/v1/balances", publicAuth)
	publicBalances.POST("", balanceTelemetry("balances.add", "balances.added", "Balance added"), gin.WrapF(handler.Add))
	publicBalances.PUT("/:customerID/:meterKey", balancePathParameters(), balanceTelemetry("balances.set", "balances.set", "Balance set"), gin.WrapF(handler.Set))

	dashboardCustomers := router.Group(
		"/v1/projects/:projectID/environments/:environment/customers",
		balanceDashboardPathParameters(),
	)
	dashboardCustomers.GET("/:customerID/balances", balanceTelemetry("balances.dashboard.list", "balances.dashboard.listed", "Customer balances loaded"), gin.WrapF(handler.List))
	dashboardCustomers.GET("/:customerID/balances/:meterKey", balanceTelemetry("balances.dashboard.get", "balances.dashboard.loaded", "Balance loaded"), gin.WrapF(handler.Get))
	dashboardCustomers.GET("/:customerID/balances/:meterKey/grants", balanceTelemetry("balances.dashboard.grants.list", "balances.dashboard.grants.listed", "Balance grants loaded"), gin.WrapF(handler.ListGrants))
	dashboardCustomers.GET("/:customerID/balances/:meterKey/history", balanceTelemetry("balances.dashboard.history.list", "balances.dashboard.history.listed", "Balance history loaded"), gin.WrapF(handler.ListActivity))
	dashboardBalances := router.Group(
		"/v1/projects/:projectID/environments/:environment/balances",
		balanceDashboardPathParameters(),
	)
	dashboardBalances.POST("", balanceTelemetry("balances.dashboard.add", "balances.dashboard.added", "Balance added"), gin.WrapF(handler.Add))
	dashboardBalances.PUT("/:customerID/:meterKey", balanceTelemetry("balances.dashboard.set", "balances.dashboard.set", "Balance set"), gin.WrapF(handler.Set))
}

func (handler *Handler) Add(response http.ResponseWriter, request *http.Request) {
	environmentID, source, ok := handler.mutationContext(response, request)
	if !ok {
		return
	}
	var input consumptionmodels.AddBalanceRequest
	if !decodeBalanceRequest(response, request, &input) {
		return
	}
	balance, replayed, err := handler.service.Add(
		request.Context(), environmentID, source, request.Header.Get(idempotencyKeyHeader), input,
	)
	if err != nil {
		handler.writeBalanceError(response, request, err)
		return
	}
	if replayed {
		response.Header().Set(idempotencyReplayHeader, "true")
	}
	handler.addLogContext(request, balance, replayed)
	if err := httpresponse.WriteJSON(response, http.StatusOK, balanceJSON(balance)); err != nil {
		handler.logResponseFailure(request, err)
	}
}

func (handler *Handler) Set(response http.ResponseWriter, request *http.Request) {
	environmentID, source, ok := handler.mutationContext(response, request)
	if !ok {
		return
	}
	var input consumptionmodels.SetBalanceRequest
	if !decodeBalanceRequest(response, request, &input) {
		return
	}
	balance, err := handler.service.Set(
		request.Context(), environmentID, source,
		request.PathValue("customerID"), request.PathValue("meterKey"), input,
	)
	if err != nil {
		handler.writeBalanceError(response, request, err)
		return
	}
	handler.addLogContext(request, balance, false)
	if err := httpresponse.WriteJSON(response, http.StatusOK, balanceJSON(balance)); err != nil {
		handler.logResponseFailure(request, err)
	}
}

func (handler *Handler) mutationContext(response http.ResponseWriter, request *http.Request) (uuid.UUID, consumptionmodels.BalanceMutationSource, bool) {
	if resolved, ok := authenticationhandlers.ProjectAPIKeyContext(request.Context()); ok {
		return resolved.ProjectEnvironmentID, consumptionmodels.BalanceMutationSource{
			Type: consumptionmodels.BalanceSourceAPIKey, ActorID: resolved.APIKeyID,
		}, true
	}
	resolved, ok := resolveDashboardEnvironmentContext(
		response, request, handler.resolver, handler.authorizer,
		"Activate Live before managing its balances.",
		func(err error) { handler.fail(response, request, err) },
	)
	if !ok {
		return uuid.Nil, consumptionmodels.BalanceMutationSource{}, false
	}
	return resolved.EnvironmentID, consumptionmodels.BalanceMutationSource{
		Type: consumptionmodels.BalanceSourceDashboardUser, ActorID: resolved.UserID,
	}, true
}

func (handler *Handler) Get(response http.ResponseWriter, request *http.Request) {
	environmentID, ok := handler.environmentID(response, request)
	if !ok {
		return
	}
	balance, err := handler.service.Get(
		request.Context(), environmentID,
		request.PathValue("customerID"), request.PathValue("meterKey"),
	)
	if err != nil {
		handler.writeBalanceError(response, request, err)
		return
	}
	handler.addLogContext(request, balance, false)
	if err := httpresponse.WriteJSON(response, http.StatusOK, balanceJSON(balance)); err != nil {
		handler.logResponseFailure(request, err)
	}
}

func (handler *Handler) List(response http.ResponseWriter, request *http.Request) {
	environmentID, ok := handler.environmentID(response, request)
	if !ok {
		return
	}
	balances, err := handler.service.List(request.Context(), environmentID, request.PathValue("customerID"))
	if err != nil {
		handler.writeBalanceError(response, request, err)
		return
	}
	telemetry.AddRequestLogAttributes(request.Context(), slog.Int("balance_count", len(balances)))
	if err := httpresponse.WriteJSON(response, http.StatusOK, balanceListJSON(balances)); err != nil {
		handler.logResponseFailure(request, err)
	}
}

func (handler *Handler) ListGrants(response http.ResponseWriter, request *http.Request) {
	environmentID, ok := handler.environmentID(response, request)
	if !ok {
		return
	}
	cursor, err := consumptionmodels.DecodeOperationCursor(request.URL.Query().Get("cursor"))
	if err != nil {
		_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_cursor", "Use the next cursor returned by Consumel.")
		return
	}
	limit, err := balanceDetailListLimit(request.URL.Query().Get("limit"))
	if err != nil {
		_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_request", "Limit must be a number between 1 and 100.")
		return
	}
	grants, next, err := handler.service.ListGrants(
		request.Context(), environmentID, request.PathValue("customerID"), request.PathValue("meterKey"), cursor, limit,
	)
	if err != nil {
		handler.writeBalanceError(response, request, err)
		return
	}
	telemetry.AddRequestLogAttributes(request.Context(), slog.Int("grant_count", len(grants)))
	if err := httpresponse.WriteJSON(response, http.StatusOK, entitlementGrantListJSON(grants, next)); err != nil {
		handler.logResponseFailure(request, err)
	}
}

func (handler *Handler) ListActivity(response http.ResponseWriter, request *http.Request) {
	environmentID, ok := handler.environmentID(response, request)
	if !ok {
		return
	}
	cursor, err := consumptionmodels.DecodeOperationCursor(request.URL.Query().Get("cursor"))
	if err != nil {
		_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_cursor", "Use the next cursor returned by Consumel.")
		return
	}
	limit, err := balanceDetailListLimit(request.URL.Query().Get("limit"))
	if err != nil {
		_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_request", "Limit must be a number between 1 and 100.")
		return
	}
	activity, next, err := handler.service.ListActivity(request.Context(), environmentID, request.PathValue("customerID"), request.PathValue("meterKey"), cursor, limit)
	if err != nil {
		handler.writeBalanceError(response, request, err)
		return
	}
	telemetry.AddRequestLogAttributes(request.Context(), slog.Int("activity_count", len(activity)))
	if err := httpresponse.WriteJSON(response, http.StatusOK, balanceActivityListJSON(activity, next)); err != nil {
		handler.logResponseFailure(request, err)
	}
}

func (handler *Handler) environmentID(response http.ResponseWriter, request *http.Request) (uuid.UUID, bool) {
	if resolved, ok := authenticationhandlers.ProjectAPIKeyContext(request.Context()); ok {
		return resolved.ProjectEnvironmentID, true
	}
	return resolveDashboardEnvironment(
		response, request, handler.resolver, handler.authorizer,
		"Activate Live before managing its balances.",
		func(err error) { handler.fail(response, request, err) },
	)
}

func (handler *Handler) writeBalanceError(response http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(err, consumptionmodels.ErrBalanceCustomerInvalid),
		errors.Is(err, consumptionmodels.ErrBalanceMeterInvalid),
		errors.Is(err, consumptionmodels.ErrBalanceQuantityInvalid),
		errors.Is(err, consumptionmodels.ErrBalanceExpirationInvalid),
		errors.Is(err, consumptionmodels.ErrIdempotencyKeyInvalid):
		_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_balance", "Check the customer ID, meter key, quantity, expiration, and idempotency key.")
	case errors.Is(err, consumptionmodels.ErrBalanceSubjectNotFound):
		_ = httpresponse.WriteError(response, http.StatusNotFound, "balance_subject_not_found", "The customer or active meter does not exist in this environment.")
	case errors.Is(err, consumptionmodels.ErrBalanceNotFound):
		_ = httpresponse.WriteError(response, http.StatusNotFound, "balance_not_found", "This customer does not have a balance for the selected meter.")
	case errors.Is(err, consumptionmodels.ErrIdempotencyKeyConflict):
		_ = httpresponse.WriteError(response, http.StatusConflict, "idempotency_key_conflict", "Use a new idempotency key for a different balance request.")
	case errors.Is(err, consumptionmodels.ErrBalanceOverflow):
		_ = httpresponse.WriteError(response, http.StatusConflict, "balance_limit_exceeded", "The resulting balance exceeds the supported quantity range.")
	default:
		handler.fail(response, request, err)
	}
}

func (handler *Handler) fail(response http.ResponseWriter, request *http.Request, err error) {
	handler.logger.ErrorContext(request.Context(), "Could not complete balance request",
		"event", "balances.request.failed", "operation", "balances.request",
		"outcome", "error", "error", err,
	)
	_ = httpresponse.WriteError(response, http.StatusInternalServerError, "balance_operation_failed", "Consumel could not complete the balance request. Try again shortly.")
}

func (handler *Handler) addLogContext(request *http.Request, balance consumptionmodels.Balance, replayed bool) {
	telemetry.AddRequestLogAttributes(request.Context(),
		slog.String("balance_id", balance.ID.String()),
		slog.Bool("idempotency_replayed", replayed),
	)
}

func (handler *Handler) logResponseFailure(request *http.Request, err error) {
	handler.logger.ErrorContext(request.Context(), "Could not send balance response",
		"event", "balances.response.failed", "operation", "balances.respond",
		"outcome", "error", "error", err,
	)
}

func decodeBalanceRequest(response http.ResponseWriter, request *http.Request, destination any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(response, request.Body, maximumBalanceRequestBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_request", "Send one valid balance JSON object.")
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_request", "Send one JSON object.")
		return false
	}
	return true
}

func balanceDetailListLimit(value string) (int, error) {
	if strings.TrimSpace(value) == "" {
		return consumptionservices.DefaultBalanceDetailPageSize, nil
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit < 1 || limit > consumptionservices.MaximumBalanceDetailPageSize {
		return 0, errors.New("invalid balance detail page size")
	}
	return limit, nil
}
