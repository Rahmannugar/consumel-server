package openapi

import (
	"github.com/google/uuid"
	"time"
)

// @Summary Return the signed-in user's active projects and environments.
// @Tags Projects
// @Success 200 {object} Projects "Completed successfully."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 500 {object} ProjectsLoadFailed "The signed-in user's projects could not be loaded."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /v1/projects [get]
func GetV1Projects() {}

// @Summary Create a project with isolated Sandbox and Live environments.
// @Tags Projects
// @Param body body models.CreateProjectRequest true "Project name."
// @Success 201 {object} Project "Created."
// @Failure 400 {object} ProjectCreateInvalid "The project name is invalid."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 409 {object} ProjectCreateConflict "The project cannot be created in the current organization state."
// @Failure 500 {object} ProjectCreateFailed "The project could not be created."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /v1/projects [post]
func PostV1Projects() {}

// @Summary Activate the selected project environment explicitly.
// @Tags Projects
// @Param projectId path string true "The immutable ID of the project selected in the dashboard." format(uuid)
// @Param environment path string true "The selected isolated project environment." enums(sandbox, live)
// @Success 200 {object} ProjectEnvironment "Completed successfully."
// @Failure 400 {object} ProjectAPIKeyInvalid "The project ID or environment is invalid."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 404 {object} ProjectAPIKeyNotFound "The project environment or active API key is unavailable."
// @Failure 500 {object} ProjectAPIKeyFailed "The API key action could not be completed."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /v1/projects/{projectId}/environments/{environment}/activate [post]
func PostV1ProjectsProjectIdEnvironmentsEnvironmentActivate() {}

// @Summary Revoke the environment's active API key.
// @Tags Projects
// @Param projectId path string true "The immutable ID of the project selected in the dashboard." format(uuid)
// @Param environment path string true "The selected isolated project environment." enums(sandbox, live)
// @Success 204 "Completed without a response body."
// @Failure 400 {object} ProjectAPIKeyInvalid "The project ID or environment is invalid."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 404 {object} ProjectAPIKeyNotFound "The project environment or active API key is unavailable."
// @Failure 500 {object} ProjectAPIKeyFailed "The API key action could not be completed."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /v1/projects/{projectId}/environments/{environment}/api-key [delete]
func DeleteV1ProjectsProjectIdEnvironmentsEnvironmentApiKey() {}

// @Summary Return safe metadata for the environment's active API key.
// @Tags Projects
// @Param projectId path string true "The immutable ID of the project selected in the dashboard." format(uuid)
// @Param environment path string true "The selected isolated project environment." enums(sandbox, live)
// @Success 200 {object} ProjectAPIKeyStatus "Completed successfully."
// @Failure 400 {object} ProjectAPIKeyInvalid "The project ID or environment is invalid."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 404 {object} ProjectAPIKeyNotFound "The project environment or active API key is unavailable."
// @Failure 500 {object} ProjectAPIKeyFailed "The API key action could not be completed."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /v1/projects/{projectId}/environments/{environment}/api-key [get]
func GetV1ProjectsProjectIdEnvironmentsEnvironmentApiKey() {}

// @Summary Create the environment's first active API key and reveal its plaintext once.
// @Tags Projects
// @Param projectId path string true "The immutable ID of the project selected in the dashboard." format(uuid)
// @Param environment path string true "The selected isolated project environment." enums(sandbox, live)
// @Success 201 {object} ProjectAPIKeyCreated "Created."
// @Failure 400 {object} ProjectAPIKeyInvalid "The project ID or environment is invalid."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 404 {object} ProjectAPIKeyNotFound "The project environment or active API key is unavailable."
// @Failure 409 {object} ProjectAPIKeyConflict "The requested API key action conflicts with the environment state."
// @Failure 500 {object} ProjectAPIKeyFailed "The API key action could not be completed."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /v1/projects/{projectId}/environments/{environment}/api-key [post]
func PostV1ProjectsProjectIdEnvironmentsEnvironmentApiKey() {}

// @Summary Revoke the active API key and reveal its replacement once.
// @Tags Projects
// @Param projectId path string true "The immutable ID of the project selected in the dashboard." format(uuid)
// @Param environment path string true "The selected isolated project environment." enums(sandbox, live)
// @Success 201 {object} ProjectAPIKeyCreated "Created."
// @Failure 400 {object} ProjectAPIKeyInvalid "The project ID or environment is invalid."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 404 {object} ProjectAPIKeyNotFound "The project environment or active API key is unavailable."
// @Failure 409 {object} ProjectAPIKeyConflict "The requested API key action conflicts with the environment state."
// @Failure 500 {object} ProjectAPIKeyFailed "The API key action could not be completed."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /v1/projects/{projectId}/environments/{environment}/api-key/replace [post]
func PostV1ProjectsProjectIdEnvironmentsEnvironmentApiKeyReplace() {}

type Project struct {
	CreatedAt        time.Time            `json:"createdAt" validate:"required" example:"2026-09-25T12:08:00Z" format:"date-time"`
	Environments     []ProjectEnvironment `json:"environments" validate:"required,max=2,min=2"`
	ID               string               `json:"id" validate:"required" example:"0199a417-05da-7aa2-b024-2011f24972da"`
	Name             string               `json:"name" validate:"required" example:"Acme API"`
	OrganizationID   string               `json:"organizationId" validate:"required" example:"0199a416-d2c8-75ea-bdb4-1d13c627169b"`
	OrganizationName string               `json:"organizationName" validate:"required" example:"Acme"`
	Slug             string               `json:"slug" validate:"required,max=120,min=1" example:"acme-api"`
}

type ProjectAPIKey struct {
	CreatedAt   time.Time  `json:"createdAt" validate:"required" example:"2026-09-27T12:00:00Z" format:"date-time"`
	Environment string     `json:"environment" validate:"required" example:"sandbox" enums:"sandbox,live"`
	ID          uuid.UUID  `json:"id" validate:"required" example:"0199a7e1-8f18-7b6e-90c9-dc7b4ace22d1" format:"uuid"`
	LastFour    string     `json:"lastFour" validate:"required,max=4,min=4" example:"QBY0"`
	LastUsedAt  *time.Time `json:"lastUsedAt" validate:"required" format:"date-time"`
	Prefix      string     `json:"prefix" validate:"required" example:"cm_test_" enums:"cm_test_,cm_live_"`
}

type ProjectAPIKeyCreated struct {
	ApiKey ProjectAPIKey `json:"apiKey" validate:"required"`
	Secret string        `json:"secret" validate:"required" example:"cm_test_3xKq7VfJm2zY8wN4aBcD6eFgH9iLpQrStUvWx0Z1A2B" pattern:"^cm_(test|live)_[A-Za-z0-9_-]{43}$"`
}

type ProjectAPIKeyStatus struct {
	ApiKey *ProjectAPIKey `json:"apiKey" validate:"required"`
}

type ProjectEnvironment struct {
	ActivatedAt *time.Time `json:"activatedAt" validate:"required" example:"2026-09-25T12:08:00Z" format:"date-time"`
	ID          string     `json:"id" validate:"required" example:"0199a417-1ae1-7b67-ad5b-809be2f9ca0a"`
	Name        string     `json:"name" validate:"required" example:"sandbox" enums:"sandbox,live"`
}

type Projects struct {
	Projects []Project `json:"projects" validate:"required"`
}

// @description Possible codes: api_key_already_exists, environment_inactive.
type ProjectAPIKeyConflict struct {
	Error ProjectAPIKeyConflictError `json:"error" validate:"required"`
}

type ProjectAPIKeyConflictError struct {
	Code    string `json:"code" validate:"required" example:"api_key_already_exists"`
	Message string `json:"message,omitempty"`
}

type ProjectAPIKeyFailed struct {
	Error ProjectAPIKeyFailedError `json:"error" validate:"required"`
}

type ProjectAPIKeyFailedError struct {
	Code    string `json:"code" validate:"required" example:"api_key_operation_failed"`
	Message string `json:"message,omitempty" example:"Consumel could not complete the API key action. Try again shortly."`
}

type ProjectAPIKeyInvalid struct {
	Error ProjectAPIKeyInvalidError `json:"error" validate:"required"`
}

type ProjectAPIKeyInvalidError struct {
	Code    string `json:"code" validate:"required" example:"invalid_request"`
	Message string `json:"message,omitempty" example:"Choose a valid project and environment."`
}

// @description Possible codes: api_key_not_found, project_environment_not_found.
type ProjectAPIKeyNotFound struct {
	Error ProjectAPIKeyNotFoundError `json:"error" validate:"required"`
}

type ProjectAPIKeyNotFoundError struct {
	Code    string `json:"code" validate:"required" example:"api_key_not_found"`
	Message string `json:"message,omitempty"`
}

// @description Possible codes: organization_required, project_name_exists.
type ProjectCreateConflict struct {
	Error ProjectCreateConflictError `json:"error" validate:"required"`
}

type ProjectCreateConflictError struct {
	Code    string `json:"code" validate:"required" example:"organization_required"`
	Message string `json:"message,omitempty"`
}

type ProjectCreateFailed struct {
	Error ProjectCreateFailedError `json:"error" validate:"required"`
}

type ProjectCreateFailedError struct {
	Code    string `json:"code" validate:"required" example:"project_create_failed"`
	Message string `json:"message,omitempty" example:"Consumel could not create the project. Try again shortly."`
}

type ProjectCreateInvalid struct {
	Error ProjectCreateInvalidError `json:"error" validate:"required"`
}

type ProjectCreateInvalidError struct {
	Code    string `json:"code" validate:"required" example:"invalid_request"`
	Message string `json:"message,omitempty" example:"Enter a project name between 1 and 120 characters."`
}

type ProjectsLoadFailed struct {
	Error ProjectsLoadFailedError `json:"error" validate:"required"`
}

type ProjectsLoadFailedError struct {
	Code    string `json:"code" validate:"required" example:"projects_load_failed"`
	Message string `json:"message,omitempty" example:"Consumel could not load your projects. Try again shortly."`
}
