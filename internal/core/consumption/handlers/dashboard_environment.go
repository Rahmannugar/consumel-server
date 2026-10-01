package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	authenticationhandlers "github.com/Rahmannugar/consumel-server/internal/authentication/handlers"
	authenticationservices "github.com/Rahmannugar/consumel-server/internal/authentication/services"
	"github.com/Rahmannugar/consumel-server/internal/common/httpresponse"
	"github.com/Rahmannugar/consumel-server/internal/infra/telemetry"
	projectmodels "github.com/Rahmannugar/consumel-server/internal/projects/models"
	"github.com/google/uuid"
)

func resolveDashboardEnvironment(
	response http.ResponseWriter,
	request *http.Request,
	resolver authenticationhandlers.TenantResolver,
	authorizer ProjectEnvironmentAuthorizer,
	inactiveMessage string,
	fail func(error),
) (uuid.UUID, bool) {
	resolved, ok := resolveDashboardEnvironmentContext(response, request, resolver, authorizer, inactiveMessage, fail)
	return resolved.EnvironmentID, ok
}

type dashboardEnvironmentContext struct {
	EnvironmentID uuid.UUID
	UserID        uuid.UUID
}

func resolveDashboardEnvironmentContext(
	response http.ResponseWriter,
	request *http.Request,
	resolver authenticationhandlers.TenantResolver,
	authorizer ProjectEnvironmentAuthorizer,
	inactiveMessage string,
	fail func(error),
) (dashboardEnvironmentContext, bool) {
	tenant, err := resolver.Resolve(request)
	if err != nil {
		if errors.Is(err, authenticationservices.ErrUnauthenticated) {
			_ = httpresponse.WriteError(response, http.StatusUnauthorized, "not_authenticated", "Sign in to access this project environment.")
			return dashboardEnvironmentContext{}, false
		}
		fail(err)
		return dashboardEnvironmentContext{}, false
	}
	projectID, err := uuid.Parse(request.PathValue("projectID"))
	if err != nil {
		_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_request", "Choose a valid project.")
		return dashboardEnvironmentContext{}, false
	}
	environmentName := projectmodels.ProjectEnvironmentName(request.PathValue("environment"))
	if environmentName != projectmodels.ProjectEnvironmentSandbox && environmentName != projectmodels.ProjectEnvironmentLive {
		_ = httpresponse.WriteError(response, http.StatusBadRequest, "invalid_request", "Choose Sandbox or Live.")
		return dashboardEnvironmentContext{}, false
	}
	environment, err := authorizer.AccessibleEnvironment(request.Context(), tenant.User.ID, projectID, environmentName)
	if errors.Is(err, projectmodels.ErrProjectEnvironmentUnavailable) {
		_ = httpresponse.WriteError(response, http.StatusNotFound, "project_environment_not_found", "This project environment is not available.")
		return dashboardEnvironmentContext{}, false
	}
	if err != nil {
		fail(err)
		return dashboardEnvironmentContext{}, false
	}
	if environment.ActivatedAt == nil {
		_ = httpresponse.WriteError(response, http.StatusConflict, "environment_inactive", inactiveMessage)
		return dashboardEnvironmentContext{}, false
	}
	telemetry.AddRequestLogAttributes(request.Context(),
		slog.String("project_id", projectID.String()),
		slog.String("project_environment_id", environment.ID.String()),
		slog.String("environment", string(environment.Name)),
	)
	return dashboardEnvironmentContext{EnvironmentID: environment.ID, UserID: tenant.User.ID}, true
}
