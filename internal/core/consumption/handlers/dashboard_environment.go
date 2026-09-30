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
	tenant, err := resolver.Resolve(request)
	if err != nil {
		if errors.Is(err, authenticationservices.ErrUnauthenticated) {
			_ = httpresponse.WriteError(response, http.StatusUnauthorized, "not_authenticated", "Sign in to access this project environment.")
			return uuid.Nil, false
		}
		fail(err)
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
	environment, err := authorizer.AccessibleEnvironment(request.Context(), tenant.User.ID, projectID, environmentName)
	if errors.Is(err, projectmodels.ErrProjectEnvironmentUnavailable) {
		_ = httpresponse.WriteError(response, http.StatusNotFound, "project_environment_not_found", "This project environment is not available.")
		return uuid.Nil, false
	}
	if err != nil {
		fail(err)
		return uuid.Nil, false
	}
	if environment.ActivatedAt == nil {
		_ = httpresponse.WriteError(response, http.StatusConflict, "environment_inactive", inactiveMessage)
		return uuid.Nil, false
	}
	telemetry.AddRequestLogAttributes(request.Context(),
		slog.String("project_id", projectID.String()),
		slog.String("project_environment_id", environment.ID.String()),
		slog.String("environment", string(environment.Name)),
	)
	return environment.ID, true
}
