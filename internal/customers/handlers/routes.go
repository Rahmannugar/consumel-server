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
	authenticationservices "github.com/Rahmannugar/consumel-server/internal/authentication/services"
	"github.com/Rahmannugar/consumel-server/internal/common/httpresponse"
	customermodels "github.com/Rahmannugar/consumel-server/internal/customers/models"
	customerservices "github.com/Rahmannugar/consumel-server/internal/customers/services"
	"github.com/Rahmannugar/consumel-server/internal/infra/telemetry"
	projectmodels "github.com/Rahmannugar/consumel-server/internal/projects/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const maximumCustomerRequestBytes = 16 << 10

type CustomerService interface {
	Create(context.Context, uuid.UUID, customermodels.CreateCustomerRequest) (customermodels.Customer, error)
	Get(context.Context, uuid.UUID, string) (customermodels.Customer, error)
	List(context.Context, uuid.UUID, *customermodels.ListCursor, int, string) ([]customermodels.Customer, *customermodels.ListCursor, error)
	Update(context.Context, uuid.UUID, string, customermodels.UpdateCustomerRequest) (customermodels.Customer, error)
}

type ProjectEnvironmentAuthorizer interface {
	AccessibleEnvironment(context.Context, uuid.UUID, uuid.UUID, projectmodels.ProjectEnvironmentName) (projectmodels.ProjectEnvironment, error)
}

type Handler struct {
	resolver   authenticationhandlers.TenantResolver
	authorizer ProjectEnvironmentAuthorizer
	service    CustomerService
	logger     *slog.Logger
}

func RegisterRoutes(
	router gin.IRouter,
	apiKeyAuthenticator authenticationhandlers.APIKeyAuthenticator,
	resolver authenticationhandlers.TenantResolver,
	authorizer ProjectEnvironmentAuthorizer,
	service CustomerService,
	logger *slog.Logger,
) {
	handler := &Handler{
		resolver: resolver, authorizer: authorizer, service: service, logger: logger,
	}
	public := router.Group("/v1/customers",
		authenticationhandlers.RequireProjectAPIKey(apiKeyAuthenticator, logger),
	)
	registerCustomerRoutes(public, handler)

	dashboard := router.Group("/v1/projects/:projectID/environments/:environment/customers",
		dashboardPathParameters(),
	)
	registerCustomerRoutes(dashboard, handler)
}

func registerCustomerRoutes(router *gin.RouterGroup, handler *Handler) {
	router.POST("", customerTelemetry("customers.create", "customers.created", "Customer created"), gin.WrapF(handler.Create))
	router.GET("", customerTelemetry("customers.list", "customers.listed", "Customers loaded"), gin.WrapF(handler.List))
	router.GET("/:customerID", customerPathParameter(), customerTelemetry("customers.get", "customers.loaded", "Customer loaded"), gin.WrapF(handler.Get))
	router.PUT("/:customerID", customerPathParameter(), customerTelemetry("customers.update", "customers.updated", "Customer updated"), gin.WrapF(handler.Update))
}

func (handler *Handler) Create(response http.ResponseWriter, request *http.Request) {
	environmentID, ok := handler.environmentID(response, request)
	if !ok {
		return
	}
	var input customermodels.CreateCustomerRequest
	if !decodeCustomerRequest(response, request, &input) {
		return
	}
	customer, err := handler.service.Create(request.Context(), environmentID, input)
	if err != nil {
		handler.writeCustomerError(response, request, err)
		return
	}
	handler.addLogContext(request, customer)
	if err := httpresponse.WriteJSON(response, http.StatusCreated, customerJSON(customer)); err != nil {
		handler.logResponseFailure(request, err)
	}
}

func (handler *Handler) List(response http.ResponseWriter, request *http.Request) {
	environmentID, ok := handler.environmentID(response, request)
	if !ok {
		return
	}
	cursor, err := customermodels.DecodeCursor(request.URL.Query().Get("cursor"))
	if err != nil {
		_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_cursor", "Use the next cursor returned by Consumel.")
		return
	}
	limit, err := listLimit(request.URL.Query().Get("limit"))
	if err != nil {
		_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_request", "Limit must be a number between 1 and 100.")
		return
	}
	customers, next, err := handler.service.List(
		request.Context(), environmentID, cursor, limit, request.URL.Query().Get("q"),
	)
	if err != nil {
		if errors.Is(err, customermodels.ErrCustomerSearchInvalid) {
			_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_search", "Search must be 120 characters or fewer.")
			return
		}
		handler.fail(response, request, err)
		return
	}
	telemetry.AddRequestLogAttributes(request.Context(), slog.Int("customer_count", len(customers)))
	if err := httpresponse.WriteJSON(response, http.StatusOK, customerListJSON(customers, next)); err != nil {
		handler.logResponseFailure(request, err)
	}
}

func (handler *Handler) Get(response http.ResponseWriter, request *http.Request) {
	environmentID, ok := handler.environmentID(response, request)
	if !ok {
		return
	}
	customer, err := handler.service.Get(request.Context(), environmentID, request.PathValue("customerID"))
	if err != nil {
		handler.writeCustomerError(response, request, err)
		return
	}
	handler.addLogContext(request, customer)
	if err := httpresponse.WriteJSON(response, http.StatusOK, customerJSON(customer)); err != nil {
		handler.logResponseFailure(request, err)
	}
}

func (handler *Handler) Update(response http.ResponseWriter, request *http.Request) {
	environmentID, ok := handler.environmentID(response, request)
	if !ok {
		return
	}
	var input customermodels.UpdateCustomerRequest
	if !decodeCustomerRequest(response, request, &input) {
		return
	}
	customer, err := handler.service.Update(
		request.Context(), environmentID, request.PathValue("customerID"), input,
	)
	if err != nil {
		handler.writeCustomerError(response, request, err)
		return
	}
	handler.addLogContext(request, customer)
	if err := httpresponse.WriteJSON(response, http.StatusOK, customerJSON(customer)); err != nil {
		handler.logResponseFailure(request, err)
	}
}

func (handler *Handler) environmentID(
	response http.ResponseWriter,
	request *http.Request,
) (uuid.UUID, bool) {
	if resolved, ok := authenticationhandlers.ProjectAPIKeyContext(request.Context()); ok {
		return resolved.ProjectEnvironmentID, true
	}

	tenant, err := handler.resolver.Resolve(request)
	if err != nil {
		if errors.Is(err, authenticationservices.ErrUnauthenticated) {
			_ = httpresponse.WriteError(response, http.StatusUnauthorized, "not_authenticated", "Sign in to manage customers.")
			return uuid.Nil, false
		}
		handler.fail(response, request, err)
		return uuid.Nil, false
	}
	projectID, err := uuid.Parse(request.PathValue("projectID"))
	if err != nil {
		_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_request", "Choose a valid project.")
		return uuid.Nil, false
	}
	environmentName := projectmodels.ProjectEnvironmentName(request.PathValue("environment"))
	if environmentName != projectmodels.ProjectEnvironmentSandbox &&
		environmentName != projectmodels.ProjectEnvironmentLive {
		_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_request", "Choose Sandbox or Live.")
		return uuid.Nil, false
	}
	environment, err := handler.authorizer.AccessibleEnvironment(
		request.Context(), tenant.User.ID, projectID, environmentName,
	)
	if errors.Is(err, projectmodels.ErrProjectEnvironmentUnavailable) {
		_ = httpresponse.WriteError(response, http.StatusNotFound, "project_environment_not_found", "This project environment is not available.")
		return uuid.Nil, false
	}
	if err != nil {
		handler.fail(response, request, err)
		return uuid.Nil, false
	}
	if environment.ActivatedAt == nil {
		_ = httpresponse.WriteError(response, http.StatusConflict, "environment_inactive", "Activate Live before managing its customers.")
		return uuid.Nil, false
	}
	telemetry.AddRequestLogAttributes(request.Context(),
		slog.String("project_id", projectID.String()),
		slog.String("project_environment_id", environment.ID.String()),
		slog.String("environment", string(environment.Name)),
	)
	return environment.ID, true
}

func (handler *Handler) writeCustomerError(response http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(err, customermodels.ErrCustomerIDRequired),
		errors.Is(err, customermodels.ErrCustomerIDTooLong),
		errors.Is(err, customermodels.ErrCustomerNameInvalid),
		errors.Is(err, customermodels.ErrCustomerEmailInvalid),
		errors.Is(err, customermodels.ErrCustomerMetadataInvalid):
		_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_customer", "Check the customer ID, contact fields, and supported metadata values.")
	case errors.Is(err, customermodels.ErrCustomerExists):
		_ = httpresponse.WriteError(response, http.StatusConflict, "customer_already_exists", "This customer ID already exists in the selected environment.")
	case errors.Is(err, customermodels.ErrCustomerNotFound):
		_ = httpresponse.WriteError(response, http.StatusNotFound, "customer_not_found", "This customer does not exist in the selected environment.")
	default:
		handler.fail(response, request, err)
	}
}

func (handler *Handler) fail(response http.ResponseWriter, request *http.Request, err error) {
	handler.logger.ErrorContext(request.Context(), "Could not complete customer request",
		"event", "customers.request.failed", "operation", "customers.request",
		"outcome", "error", "error", err,
	)
	_ = httpresponse.WriteError(response, http.StatusInternalServerError, "customer_operation_failed", "Consumel could not complete the customer request. Try again shortly.")
}

func (handler *Handler) addLogContext(request *http.Request, customer customermodels.Customer) {
	telemetry.AddRequestLogAttributes(request.Context(),
		slog.String("customer_record_id", customer.ID.String()),
	)
}

func (handler *Handler) logResponseFailure(request *http.Request, err error) {
	handler.logger.ErrorContext(request.Context(), "Could not send customer response",
		"event", "customers.response.failed", "operation", "customers.respond",
		"outcome", "error", "error", err,
	)
}

func decodeCustomerRequest(response http.ResponseWriter, request *http.Request, destination any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(response, request.Body, maximumCustomerRequestBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_request", "Send one valid customer JSON object.")
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_request", "Send one JSON object.")
		return false
	}
	return true
}

func listLimit(value string) (int, error) {
	if strings.TrimSpace(value) == "" {
		return customerservices.DefaultPageSize, nil
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit < 1 || limit > customerservices.MaximumPageSize {
		return 0, errors.New("invalid customer page size")
	}
	return limit, nil
}

func customerPathParameter() gin.HandlerFunc {
	return func(context *gin.Context) {
		context.Request.SetPathValue("customerID", context.Param("customerID"))
		context.Next()
	}
}

func dashboardPathParameters() gin.HandlerFunc {
	return func(context *gin.Context) {
		context.Request.SetPathValue("projectID", context.Param("projectID"))
		context.Request.SetPathValue("environment", context.Param("environment"))
		context.Next()
	}
}
