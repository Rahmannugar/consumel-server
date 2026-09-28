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
	"github.com/Rahmannugar/consumel-server/internal/infra/telemetry"
	metermodels "github.com/Rahmannugar/consumel-server/internal/meters/models"
	meterservices "github.com/Rahmannugar/consumel-server/internal/meters/services"
	projectmodels "github.com/Rahmannugar/consumel-server/internal/projects/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const maximumMeterRequestBytes = 16 << 10

type MeterService interface {
	Create(context.Context, uuid.UUID, metermodels.CreateMeterRequest) (metermodels.Meter, error)
	Get(context.Context, uuid.UUID, string) (metermodels.Meter, error)
	List(context.Context, uuid.UUID, *metermodels.ListCursor, int) ([]metermodels.Meter, *metermodels.ListCursor, error)
}

type ProjectEnvironmentAuthorizer interface {
	AccessibleEnvironment(context.Context, uuid.UUID, uuid.UUID, projectmodels.ProjectEnvironmentName) (projectmodels.ProjectEnvironment, error)
}

type Handler struct {
	resolver   authenticationhandlers.TenantResolver
	authorizer ProjectEnvironmentAuthorizer
	service    MeterService
	logger     *slog.Logger
}

func RegisterRoutes(
	router gin.IRouter,
	apiKeyAuthenticator authenticationhandlers.APIKeyAuthenticator,
	resolver authenticationhandlers.TenantResolver,
	authorizer ProjectEnvironmentAuthorizer,
	service MeterService,
	logger *slog.Logger,
) {
	handler := &Handler{resolver: resolver, authorizer: authorizer, service: service, logger: logger}
	public := router.Group("/v1/meters",
		authenticationhandlers.RequireProjectAPIKey(apiKeyAuthenticator, logger),
	)
	registerMeterRoutes(public, handler)
	dashboard := router.Group("/v1/projects/:projectID/environments/:environment/meters",
		dashboardPathParameters(),
	)
	registerMeterRoutes(dashboard, handler)
}

func registerMeterRoutes(router *gin.RouterGroup, handler *Handler) {
	router.POST("", meterTelemetry("meters.create", "meters.created", "Meter created"), gin.WrapF(handler.Create))
	router.GET("", meterTelemetry("meters.list", "meters.listed", "Meters loaded"), gin.WrapF(handler.List))
	router.GET("/:meterKey", meterPathParameter(), meterTelemetry("meters.get", "meters.loaded", "Meter loaded"), gin.WrapF(handler.Get))
}

func (handler *Handler) Create(response http.ResponseWriter, request *http.Request) {
	environmentID, ok := handler.environmentID(response, request)
	if !ok {
		return
	}
	var input metermodels.CreateMeterRequest
	if !decodeMeterRequest(response, request, &input) {
		return
	}
	meter, err := handler.service.Create(request.Context(), environmentID, input)
	if err != nil {
		handler.writeMeterError(response, request, err)
		return
	}
	handler.addLogContext(request, meter)
	if err := httpresponse.WriteJSON(response, http.StatusCreated, meterJSON(meter)); err != nil {
		handler.logResponseFailure(request, err)
	}
}

func (handler *Handler) List(response http.ResponseWriter, request *http.Request) {
	environmentID, ok := handler.environmentID(response, request)
	if !ok {
		return
	}
	cursor, err := metermodels.DecodeCursor(request.URL.Query().Get("cursor"))
	if err != nil {
		_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_cursor", "Use the next cursor returned by Consumel.")
		return
	}
	limit, err := listLimit(request.URL.Query().Get("limit"))
	if err != nil {
		_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_request", "Limit must be a number between 1 and 100.")
		return
	}
	meters, next, err := handler.service.List(request.Context(), environmentID, cursor, limit)
	if err != nil {
		handler.fail(response, request, err)
		return
	}
	telemetry.AddRequestLogAttributes(request.Context(), slog.Int("meter_count", len(meters)))
	if err := httpresponse.WriteJSON(response, http.StatusOK, meterListJSON(meters, next)); err != nil {
		handler.logResponseFailure(request, err)
	}
}

func (handler *Handler) Get(response http.ResponseWriter, request *http.Request) {
	environmentID, ok := handler.environmentID(response, request)
	if !ok {
		return
	}
	meter, err := handler.service.Get(request.Context(), environmentID, request.PathValue("meterKey"))
	if err != nil {
		handler.writeMeterError(response, request, err)
		return
	}
	handler.addLogContext(request, meter)
	if err := httpresponse.WriteJSON(response, http.StatusOK, meterJSON(meter)); err != nil {
		handler.logResponseFailure(request, err)
	}
}

func (handler *Handler) environmentID(response http.ResponseWriter, request *http.Request) (uuid.UUID, bool) {
	if resolved, ok := authenticationhandlers.ProjectAPIKeyContext(request.Context()); ok {
		return resolved.ProjectEnvironmentID, true
	}
	tenant, err := handler.resolver.Resolve(request)
	if err != nil {
		if errors.Is(err, authenticationservices.ErrUnauthenticated) {
			_ = httpresponse.WriteError(response, http.StatusUnauthorized, "not_authenticated", "Sign in to manage meters.")
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
	if environmentName != projectmodels.ProjectEnvironmentSandbox && environmentName != projectmodels.ProjectEnvironmentLive {
		_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_request", "Choose Sandbox or Live.")
		return uuid.Nil, false
	}
	environment, err := handler.authorizer.AccessibleEnvironment(request.Context(), tenant.User.ID, projectID, environmentName)
	if errors.Is(err, projectmodels.ErrProjectEnvironmentUnavailable) {
		_ = httpresponse.WriteError(response, http.StatusNotFound, "project_environment_not_found", "This project environment is not available.")
		return uuid.Nil, false
	}
	if err != nil {
		handler.fail(response, request, err)
		return uuid.Nil, false
	}
	if environment.ActivatedAt == nil {
		_ = httpresponse.WriteError(response, http.StatusConflict, "environment_inactive", "Activate Live before managing its meters.")
		return uuid.Nil, false
	}
	telemetry.AddRequestLogAttributes(request.Context(),
		slog.String("project_id", projectID.String()),
		slog.String("project_environment_id", environment.ID.String()),
		slog.String("environment", string(environment.Name)),
	)
	return environment.ID, true
}

func (handler *Handler) writeMeterError(response http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(err, metermodels.ErrMeterKeyInvalid),
		errors.Is(err, metermodels.ErrMeterNameInvalid),
		errors.Is(err, metermodels.ErrMeterDescriptionInvalid),
		errors.Is(err, metermodels.ErrMeterTypeInvalid):
		_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_meter", "Check the meter key, name, description, and type.")
	case errors.Is(err, metermodels.ErrMeterExists):
		_ = httpresponse.WriteError(response, http.StatusConflict, "meter_already_exists", "This meter key already exists in the selected environment.")
	case errors.Is(err, metermodels.ErrMeterNotFound):
		_ = httpresponse.WriteError(response, http.StatusNotFound, "meter_not_found", "This meter does not exist in the selected environment.")
	default:
		handler.fail(response, request, err)
	}
}

func (handler *Handler) fail(response http.ResponseWriter, request *http.Request, err error) {
	handler.logger.ErrorContext(request.Context(), "Could not complete meter request",
		"event", "meters.request.failed", "operation", "meters.request",
		"outcome", "error", "error", err,
	)
	_ = httpresponse.WriteError(response, http.StatusInternalServerError, "meter_operation_failed", "Consumel could not complete the meter request. Try again shortly.")
}

func (handler *Handler) addLogContext(request *http.Request, meter metermodels.Meter) {
	telemetry.AddRequestLogAttributes(request.Context(), slog.String("meter_id", meter.ID.String()))
}

func (handler *Handler) logResponseFailure(request *http.Request, err error) {
	handler.logger.ErrorContext(request.Context(), "Could not send meter response",
		"event", "meters.response.failed", "operation", "meters.respond",
		"outcome", "error", "error", err,
	)
}

func decodeMeterRequest(response http.ResponseWriter, request *http.Request, destination any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(response, request.Body, maximumMeterRequestBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_request", "Send one valid meter JSON object.")
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
		return meterservices.DefaultPageSize, nil
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit < 1 || limit > meterservices.MaximumPageSize {
		return 0, errors.New("invalid meter page size")
	}
	return limit, nil
}

func meterPathParameter() gin.HandlerFunc {
	return func(context *gin.Context) {
		context.Request.SetPathValue("meterKey", context.Param("meterKey"))
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
