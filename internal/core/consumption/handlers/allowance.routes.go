package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	authenticationhandlers "github.com/Rahmannugar/consumel-server/internal/authentication/handlers"
	"github.com/Rahmannugar/consumel-server/internal/common/httpresponse"
	consumptionmodels "github.com/Rahmannugar/consumel-server/internal/core/consumption/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AllowanceService interface {
	Periods(context.Context, uuid.UUID, string, string, *consumptionmodels.OperationListCursor, int) (consumptionmodels.AllowancePeriods, error)
	Save(context.Context, uuid.UUID, string, string, consumptionmodels.BalanceMutationSource, consumptionmodels.AllowanceRequest) (consumptionmodels.Allowance, error)
	Get(context.Context, uuid.UUID, string, string) (consumptionmodels.Allowance, error)
}

type AllowanceHandler struct {
	boundary *Handler
	service  AllowanceService
}

func RegisterAllowanceRoutes(router gin.IRouter, keys authenticationhandlers.APIKeyAuthenticator, resolver authenticationhandlers.TenantResolver, authorizer ProjectEnvironmentAuthorizer, service AllowanceService, logger *slog.Logger) {
	handler := &AllowanceHandler{boundary: &Handler{resolver: resolver, authorizer: authorizer, logger: logger}, service: service}
	public := router.Group("/v1/customers", authenticationhandlers.RequireProjectAPIKey(keys, logger), balancePathParameters())
	public.GET("/:customerID/balances/:meterKey/allowance", balanceTelemetry("allowances.get", "allowances.loaded", "Recurring allowance loaded"), gin.WrapF(handler.Get))
	public.GET("/:customerID/balances/:meterKey/allowance/periods", balanceTelemetry("allowances.periods.list", "allowances.periods.listed", "Allowance periods loaded"), gin.WrapF(handler.Periods))
	public.PUT("/:customerID/balances/:meterKey/allowance", balanceTelemetry("allowances.save", "allowances.saved", "Recurring allowance saved"), gin.WrapF(handler.Save))
	dashboard := router.Group("/v1/projects/:projectID/environments/:environment/customers", balanceDashboardPathParameters())
	dashboard.GET("/:customerID/balances/:meterKey/allowance", balanceTelemetry("allowances.dashboard.get", "allowances.dashboard.loaded", "Recurring allowance loaded"), gin.WrapF(handler.Get))
	dashboard.GET("/:customerID/balances/:meterKey/allowance/periods", balanceTelemetry("allowances.dashboard.periods.list", "allowances.dashboard.periods.listed", "Allowance periods loaded"), gin.WrapF(handler.Periods))
	dashboard.PUT("/:customerID/balances/:meterKey/allowance", balanceTelemetry("allowances.dashboard.save", "allowances.dashboard.saved", "Recurring allowance saved"), gin.WrapF(handler.Save))
}
func (h *AllowanceHandler) Get(w http.ResponseWriter, r *http.Request) {
	env, _, ok := h.boundary.mutationContext(w, r)
	if !ok {
		return
	}
	result, err := h.service.Get(r.Context(), env, r.PathValue("customerID"), r.PathValue("meterKey"))
	h.respond(w, r, result, err)
}
func (h *AllowanceHandler) Save(w http.ResponseWriter, r *http.Request) {
	env, source, ok := h.boundary.mutationContext(w, r)
	if !ok {
		return
	}
	var input consumptionmodels.AllowanceRequest
	if !decodeBalanceRequest(w, r, &input) {
		return
	}
	result, err := h.service.Save(r.Context(), env, r.PathValue("customerID"), r.PathValue("meterKey"), source, input)
	h.respond(w, r, result, err)
}
func (h *AllowanceHandler) respond(w http.ResponseWriter, r *http.Request, result consumptionmodels.Allowance, err error) {
	switch {
	case errors.Is(err, consumptionmodels.ErrAllowanceCursorInvalid):
		_ = httpresponse.WriteError(w, 400, "invalid_cursor", "Use the next cursor returned for this allowance.")
	case errors.Is(err, consumptionmodels.ErrAllowanceNotFound):
		_ = httpresponse.WriteError(w, 404, "allowance_not_found", "No recurring allowance is configured for this customer and meter.")
	case errors.Is(err, consumptionmodels.ErrAllowanceInvalid):
		_ = httpresponse.WriteError(w, 400, "invalid_allowance", "Use a positive quantity, a nonnegative rollover cap, and a start date at midnight UTC.")
	case errors.Is(err, consumptionmodels.ErrAllowanceConflict):
		_ = httpresponse.WriteError(w, 409, "allowance_version_conflict", "Reload the allowance before editing. Its start date cannot change after creation.")
	case err != nil:
		h.boundary.writeBalanceError(w, r, err)
	default:
		if err = httpresponse.WriteJSON(w, 200, result); err != nil {
			h.boundary.logResponseFailure(r, err)
		}
	}
}

func (h *AllowanceHandler) Periods(w http.ResponseWriter, r *http.Request) {
	env, _, ok := h.boundary.mutationContext(w, r)
	if !ok {
		return
	}
	cursor, err := consumptionmodels.DecodeOperationCursor(r.URL.Query().Get("cursor"))
	if err != nil {
		h.respond(w, r, consumptionmodels.Allowance{}, consumptionmodels.ErrAllowanceCursorInvalid)
		return
	}
	limit, err := balanceDetailListLimit(r.URL.Query().Get("limit"))
	if err != nil {
		h.respond(w, r, consumptionmodels.Allowance{}, consumptionmodels.ErrAllowanceInvalid)
		return
	}
	result, err := h.service.Periods(r.Context(), env, r.PathValue("customerID"), r.PathValue("meterKey"), cursor, limit)
	if err != nil {
		h.respond(w, r, consumptionmodels.Allowance{}, err)
		return
	}
	if err = httpresponse.WriteJSON(w, 200, result); err != nil {
		h.boundary.logResponseFailure(r, err)
	}
}
