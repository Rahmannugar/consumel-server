package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
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

type ProjectService interface {
	ListProjects(context.Context, uuid.UUID) ([]projectmodels.Project, error)
	CreateProject(context.Context, uuid.UUID, string) (projectmodels.Project, error)
	ActiveAPIKey(context.Context, uuid.UUID, uuid.UUID, projectmodels.ProjectEnvironmentName) (*projectmodels.APIKey, error)
	CreateAPIKey(context.Context, uuid.UUID, uuid.UUID, projectmodels.ProjectEnvironmentName) (projectmodels.CreatedAPIKey, error)
	ReplaceAPIKey(context.Context, uuid.UUID, uuid.UUID, projectmodels.ProjectEnvironmentName) (projectmodels.CreatedAPIKey, error)
	RevokeAPIKey(context.Context, uuid.UUID, uuid.UUID, projectmodels.ProjectEnvironmentName) error
	ActivateEnvironment(context.Context, uuid.UUID, uuid.UUID, projectmodels.ProjectEnvironmentName) (projectmodels.ProjectEnvironment, error)
	AccessibleEnvironment(context.Context, uuid.UUID, uuid.UUID, projectmodels.ProjectEnvironmentName) (projectmodels.ProjectEnvironment, error)
}

type Handler struct {
	resolver authenticationhandlers.TenantResolver
	service  ProjectService
	logger   *slog.Logger
}

func RegisterRoutes(
	router gin.IRouter,
	resolver authenticationhandlers.TenantResolver,
	service ProjectService,
	logger *slog.Logger,
) {
	handler := &Handler{resolver: resolver, service: service, logger: logger}
	router.GET("/v1/projects", projectListTelemetry(), gin.WrapF(handler.List))
	router.POST("/v1/projects", projectCreateTelemetry(), gin.WrapF(handler.Create))
	apiKeyPath := "/v1/projects/:projectID/environments/:environment/api-key"
	router.GET(apiKeyPath, apiKeyTelemetry("projects.api_key.get", "projects.api_key.loaded", "Project API key loaded"), pathParameters(), gin.WrapF(handler.GetAPIKey))
	router.POST(apiKeyPath, apiKeyTelemetry("projects.api_key.create", "projects.api_key.created", "Project API key created"), pathParameters(), gin.WrapF(handler.CreateAPIKey))
	router.POST(apiKeyPath+"/replace", apiKeyTelemetry("projects.api_key.replace", "projects.api_key.replaced", "Project API key replaced"), pathParameters(), gin.WrapF(handler.ReplaceAPIKey))
	router.DELETE(apiKeyPath, apiKeyTelemetry("projects.api_key.revoke", "projects.api_key.revoked", "Project API key revoked"), pathParameters(), gin.WrapF(handler.RevokeAPIKey))
	router.POST("/v1/projects/:projectID/environments/:environment/activate", apiKeyTelemetry("projects.environment.activate", "projects.environment.activated", "Project environment activated"), pathParameters(), gin.WrapF(handler.ActivateEnvironment))
}

func (handler *Handler) Create(response http.ResponseWriter, request *http.Request) {
	tenant, err := handler.resolver.Resolve(request)
	if err != nil {
		if errors.Is(err, authenticationservices.ErrUnauthenticated) {
			_ = httpresponse.WriteError(response, http.StatusUnauthorized, "not_authenticated",
				"Sign in to create a project.")
			return
		}
		handler.projectCreateFailure(response, request, err)
		return
	}
	if len(tenant.OrganizationAccess) != 1 {
		_ = httpresponse.WriteError(response, http.StatusConflict, "organization_required",
			"Finish organization setup before creating another project.")
		return
	}

	var input projectmodels.CreateProjectRequest
	decoder := json.NewDecoder(http.MaxBytesReader(response, request.Body, 16<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_request",
			"Enter a project name between 1 and 120 characters.")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_request",
			"Send one JSON object.")
		return
	}
	input, err = input.Validate()
	if err != nil {
		_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_request",
			"Enter a project name between 1 and 120 characters.")
		return
	}

	access := tenant.OrganizationAccess[0]
	project, err := handler.service.CreateProject(request.Context(), access.OrganizationID, input.Name)
	if err != nil {
		if errors.Is(err, projectmodels.ErrProjectNameExists) {
			_ = httpresponse.WriteError(response, http.StatusConflict, "project_name_exists",
				"A project with this name already exists in your organization.")
			return
		}
		handler.projectCreateFailure(response, request, err)
		return
	}
	project.OrganizationName = access.OrganizationName
	telemetry.AddRequestLogAttributes(request.Context(),
		slog.String("organization_id", access.OrganizationID.String()),
		slog.String("project_id", project.ID.String()),
	)
	if err := httpresponse.WriteJSON(response, http.StatusCreated, projectJSON(project)); err != nil {
		handler.logger.ErrorContext(request.Context(), "Could not send created project response",
			"event", "projects.create.response.failed", "operation", "projects.create.respond",
			"outcome", "error", "error", err,
		)
	}
}

func (handler *Handler) projectCreateFailure(
	response http.ResponseWriter,
	request *http.Request,
	err error,
) {
	handler.logger.ErrorContext(request.Context(), "Could not create project",
		"event", "projects.create.failed", "operation", "projects.create",
		"outcome", "error", "error", err,
	)
	_ = httpresponse.WriteError(response, http.StatusInternalServerError, "project_create_failed",
		"Consumel could not create the project. Try again shortly.")
}

func pathParameters() gin.HandlerFunc {
	return func(context *gin.Context) {
		context.Request.SetPathValue("projectID", context.Param("projectID"))
		context.Request.SetPathValue("environment", context.Param("environment"))
		context.Next()
	}
}

func (handler *Handler) List(response http.ResponseWriter, request *http.Request) {
	tenant, err := handler.resolver.Resolve(request)
	if err != nil {
		if errors.Is(err, authenticationservices.ErrUnauthenticated) {
			_ = httpresponse.WriteError(response, http.StatusUnauthorized, "not_authenticated",
				"Sign in to view your projects.")
			return
		}
		handler.fail(response, request, err)
		return
	}
	projects, err := handler.service.ListProjects(request.Context(), tenant.User.ID)
	if err != nil {
		handler.fail(response, request, err)
		return
	}
	telemetry.AddRequestLogAttributes(request.Context(),
		slog.String("user_id", tenant.User.ID.String()),
		slog.Int("project_count", len(projects)),
	)
	if err := httpresponse.WriteJSON(response, http.StatusOK, listResponse(projects)); err != nil {
		handler.logger.ErrorContext(request.Context(), "Could not send projects response",
			"event", "projects.response.failed",
			"operation", "projects.list.respond",
			"outcome", "error",
			"error", err,
		)
	}
}

func (handler *Handler) fail(response http.ResponseWriter, request *http.Request, err error) {
	handler.logger.ErrorContext(request.Context(), "Could not list projects",
		"event", "projects.list.failed",
		"operation", "projects.list",
		"outcome", "error",
		"error", err,
	)
	_ = httpresponse.WriteError(response, http.StatusInternalServerError, "projects_load_failed",
		"Consumel could not load your projects. Try again shortly.")
}

func (handler *Handler) GetAPIKey(response http.ResponseWriter, request *http.Request) {
	userID, projectID, environment, ok := handler.apiKeyContext(response, request)
	if !ok {
		return
	}
	key, err := handler.service.ActiveAPIKey(request.Context(), userID, projectID, environment)
	if err != nil {
		handler.apiKeyError(response, request, "load", err)
		return
	}
	handler.addAPIKeyLogContext(request, projectID, environment, key)
	if err := httpresponse.WriteJSON(response, http.StatusOK, apiKeyStatus(key)); err != nil {
		handler.logResponseFailure(request, "load", err)
	}
}

func (handler *Handler) CreateAPIKey(response http.ResponseWriter, request *http.Request) {
	userID, projectID, environment, ok := handler.apiKeyContext(response, request)
	if !ok {
		return
	}
	key, err := handler.service.CreateAPIKey(request.Context(), userID, projectID, environment)
	if err != nil {
		handler.apiKeyError(response, request, "create", err)
		return
	}
	handler.addAPIKeyLogContext(request, projectID, environment, &key.APIKey)
	if err := httpresponse.WriteJSON(response, http.StatusCreated, createdAPIKeyJSON(key)); err != nil {
		handler.logResponseFailure(request, "create", err)
	}
}

func (handler *Handler) ReplaceAPIKey(response http.ResponseWriter, request *http.Request) {
	userID, projectID, environment, ok := handler.apiKeyContext(response, request)
	if !ok {
		return
	}
	key, err := handler.service.ReplaceAPIKey(request.Context(), userID, projectID, environment)
	if err != nil {
		handler.apiKeyError(response, request, "replace", err)
		return
	}
	handler.addAPIKeyLogContext(request, projectID, environment, &key.APIKey)
	if err := httpresponse.WriteJSON(response, http.StatusCreated, createdAPIKeyJSON(key)); err != nil {
		handler.logResponseFailure(request, "replace", err)
	}
}

func (handler *Handler) RevokeAPIKey(response http.ResponseWriter, request *http.Request) {
	userID, projectID, environment, ok := handler.apiKeyContext(response, request)
	if !ok {
		return
	}
	if err := handler.service.RevokeAPIKey(request.Context(), userID, projectID, environment); err != nil {
		handler.apiKeyError(response, request, "revoke", err)
		return
	}
	telemetry.AddRequestLogAttributes(request.Context(),
		slog.String("project_id", projectID.String()),
		slog.String("environment", string(environment)),
	)
	response.WriteHeader(http.StatusNoContent)
}

func (handler *Handler) ActivateEnvironment(response http.ResponseWriter, request *http.Request) {
	userID, projectID, environment, ok := handler.apiKeyContext(response, request)
	if !ok {
		return
	}
	activated, err := handler.service.ActivateEnvironment(
		request.Context(), userID, projectID, environment,
	)
	if err != nil {
		handler.apiKeyError(response, request, "activate_environment", err)
		return
	}
	telemetry.AddRequestLogAttributes(request.Context(),
		slog.String("project_id", projectID.String()),
		slog.String("environment", string(environment)),
	)
	if err := httpresponse.WriteJSON(response, http.StatusOK, environmentResponse{
		ID: activated.ID.String(), Name: string(activated.Name), ActivatedAt: activated.ActivatedAt,
	}); err != nil {
		handler.logResponseFailure(request, "activate_environment", err)
	}
}

func (handler *Handler) apiKeyContext(
	response http.ResponseWriter,
	request *http.Request,
) (uuid.UUID, uuid.UUID, projectmodels.ProjectEnvironmentName, bool) {
	tenant, err := handler.resolver.Resolve(request)
	if err != nil {
		if errors.Is(err, authenticationservices.ErrUnauthenticated) {
			_ = httpresponse.WriteError(response, http.StatusUnauthorized, "not_authenticated", "Sign in to manage API keys.")
			return uuid.Nil, uuid.Nil, "", false
		}
		handler.apiKeyError(response, request, "authorize", err)
		return uuid.Nil, uuid.Nil, "", false
	}
	projectID, err := uuid.Parse(request.PathValue("projectID"))
	if err != nil {
		_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_request", "Choose a valid project.")
		return uuid.Nil, uuid.Nil, "", false
	}
	environment := projectmodels.ProjectEnvironmentName(request.PathValue("environment"))
	if environment != projectmodels.ProjectEnvironmentSandbox && environment != projectmodels.ProjectEnvironmentLive {
		_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_request", "Choose Sandbox or Live.")
		return uuid.Nil, uuid.Nil, "", false
	}
	return tenant.User.ID, projectID, environment, true
}

func (handler *Handler) apiKeyError(
	response http.ResponseWriter,
	request *http.Request,
	action string,
	err error,
) {
	switch {
	case errors.Is(err, projectmodels.ErrProjectEnvironmentUnavailable):
		_ = httpresponse.WriteError(response, http.StatusNotFound, "project_environment_not_found", "This project environment is not available.")
	case errors.Is(err, projectmodels.ErrProjectEnvironmentInactive):
		_ = httpresponse.WriteError(response, http.StatusConflict, "environment_inactive", "Activate Live before creating its API key.")
	case errors.Is(err, projectmodels.ErrActiveAPIKeyExists):
		_ = httpresponse.WriteError(response, http.StatusConflict, "api_key_already_exists", "This environment already has an active API key. Replace it to issue a new key.")
	case errors.Is(err, projectmodels.ErrActiveAPIKeyNotFound):
		_ = httpresponse.WriteError(response, http.StatusNotFound, "api_key_not_found", "This environment does not have an active API key.")
	default:
		handler.logger.ErrorContext(request.Context(), "Could not manage project API key",
			"event", "projects.api_key."+action+".failed",
			"operation", "projects.api_key."+action,
			"outcome", "error",
			"error", err,
		)
		_ = httpresponse.WriteError(response, http.StatusInternalServerError, "api_key_operation_failed", "Consumel could not complete the API key action. Try again shortly.")
	}
}

func (handler *Handler) addAPIKeyLogContext(
	request *http.Request,
	projectID uuid.UUID,
	environment projectmodels.ProjectEnvironmentName,
	key *projectmodels.APIKey,
) {
	attributes := []slog.Attr{
		slog.String("project_id", projectID.String()),
		slog.String("environment", string(environment)),
	}
	if key != nil {
		attributes = append(attributes, slog.String("api_key_id", key.ID.String()))
	}
	telemetry.AddRequestLogAttributes(request.Context(), attributes...)
}

func (handler *Handler) logResponseFailure(request *http.Request, action string, err error) {
	handler.logger.ErrorContext(request.Context(), "Could not send project API key response",
		"event", "projects.api_key."+action+".response.failed",
		"operation", "projects.api_key."+action+".respond",
		"outcome", "error",
		"error", err,
	)
}
