package openapi

import (
	"github.com/google/uuid"
	"time"
)

// @Summary List meters in the API key's project environment.
// @Tags Meters
// @Param q query string false "A case-insensitive meter key or name search." maxLength(120)
// @Param cursor query string false "The opaque next cursor from the previous page."
// @Param limit query int false "The number of meters to return." minimum(1) maximum(100) default(25)
// @Success 200 {object} Meters "Completed successfully."
// @Failure 400 {object} MeterListInvalid "The list pagination or search input is invalid."
// @Failure 401 {object} InvalidAPIKey "The project API key is missing, malformed, revoked, replaced, or inactive."
// @Failure 500 {object} MeterFailed "The meter request could not be completed."
// @Security projectAPIKey
// @Router /v1/meters [get]
func GetV1Meters() {}

// @Summary Create a meter in the API key's project environment.
// @Tags Meters
// @Param body body models.CreateMeterRequest true "Meter definition."
// @Success 201 {object} Meter "Created."
// @Failure 400 {object} MeterInvalid "The meter request or key is invalid."
// @Failure 401 {object} InvalidAPIKey "The project API key is missing, malformed, revoked, replaced, or inactive."
// @Failure 409 {object} MeterConflict "The meter cannot be created in the API key's environment."
// @Failure 500 {object} MeterFailed "The meter request could not be completed."
// @Security projectAPIKey
// @Router /v1/meters [post]
func PostV1Meters() {}

// @Summary Return one meter from the API key's project environment.
// @Tags Meters
// @Param meterKey path string true "The stable meter key supplied when the meter was created. Must match `^[a-z][a-z0-9_-]*$`." maxLength(120)
// @Success 200 {object} Meter "Completed successfully."
// @Failure 400 {object} MeterInvalid "The meter request or key is invalid."
// @Failure 401 {object} InvalidAPIKey "The project API key is missing, malformed, revoked, replaced, or inactive."
// @Failure 404 {object} MeterNotFound "The meter is not present in the authenticated environment."
// @Failure 500 {object} MeterFailed "The meter request could not be completed."
// @Security projectAPIKey
// @Router /v1/meters/{meterKey} [get]
func GetV1MetersMeterKey() {}

// @Summary List meters in the signed-in project workspace.
// @Tags Dashboard Meters
// @Param projectId path string true "The immutable ID of the project selected in the dashboard." format(uuid)
// @Param environment path string true "The selected isolated project environment." enums(sandbox, live)
// @Param q query string false "A case-insensitive meter key or name search." maxLength(120)
// @Param cursor query string false "The opaque next cursor from the previous page."
// @Param limit query int false "The number of meters to return." minimum(1) maximum(100) default(25)
// @Success 200 {object} Meters "Completed successfully."
// @Failure 400 {object} MeterListInvalid "The list pagination or search input is invalid."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 404 {object} MeterEnvironmentNotFound "The selected project environment is unavailable."
// @Failure 409 {object} MeterEnvironmentConflict "The selected project environment is not active."
// @Failure 500 {object} MeterFailed "The meter request could not be completed."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /v1/projects/{projectId}/environments/{environment}/meters [get]
func GetV1ProjectsProjectIdEnvironmentsEnvironmentMeters() {}

// @Summary Create a meter from the signed-in project workspace.
// @Tags Dashboard Meters
// @Param projectId path string true "The immutable ID of the project selected in the dashboard." format(uuid)
// @Param environment path string true "The selected isolated project environment." enums(sandbox, live)
// @Param body body models.CreateMeterRequest true "Meter definition."
// @Success 201 {object} Meter "Created."
// @Failure 400 {object} MeterInvalid "The meter request or key is invalid."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 404 {object} MeterEnvironmentNotFound "The selected project environment is unavailable."
// @Failure 409 {object} MeterCreateConflict "The meter cannot be created in the selected environment."
// @Failure 500 {object} MeterFailed "The meter request could not be completed."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /v1/projects/{projectId}/environments/{environment}/meters [post]
func PostV1ProjectsProjectIdEnvironmentsEnvironmentMeters() {}

// @Summary Return one meter from the signed-in project workspace.
// @Tags Dashboard Meters
// @Param projectId path string true "The immutable ID of the project selected in the dashboard." format(uuid)
// @Param environment path string true "The selected isolated project environment." enums(sandbox, live)
// @Param meterKey path string true "The stable meter key supplied when the meter was created. Must match `^[a-z][a-z0-9_-]*$`." maxLength(120)
// @Success 200 {object} Meter "Completed successfully."
// @Failure 400 {object} MeterInvalid "The meter request or key is invalid."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 404 {object} MeterOrEnvironmentNotFound "The meter or selected project environment is unavailable."
// @Failure 409 {object} MeterEnvironmentConflict "The selected project environment is not active."
// @Failure 500 {object} MeterFailed "The meter request could not be completed."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /v1/projects/{projectId}/environments/{environment}/meters/{meterKey} [get]
func GetV1ProjectsProjectIdEnvironmentsEnvironmentMetersMeterKey() {}

type Meter struct {
	CreatedAt   time.Time `json:"createdAt" validate:"required" example:"2026-09-28T12:00:00Z" format:"date-time"`
	Description *string   `json:"description" validate:"required,max=500" example:"Requests processed by your API."`
	ID          uuid.UUID `json:"id" validate:"required" example:"0199a9f8-f0c4-7f10-90f8-6483353e4624" format:"uuid"`
	MeterKey    string    `json:"meterKey" validate:"required,max=120" example:"api_calls" pattern:"^[a-z][a-z0-9_-]*$"`
	Name        string    `json:"name" validate:"required,max=120" example:"API calls"`
	Type        string    `json:"type" validate:"required" example:"postpaid" enums:"prepaid,postpaid,hybrid"`
	UpdatedAt   time.Time `json:"updatedAt" validate:"required" example:"2026-09-28T12:00:00Z" format:"date-time"`
}

type Meters struct {
	Meters     []Meter `json:"meters" validate:"required"`
	NextCursor *string `json:"nextCursor" validate:"required"`
}

// @description Possible codes: meter_already_exists, meter_definition_conflict.
type MeterConflict struct {
	Error MeterConflictError `json:"error" validate:"required"`
}

type MeterConflictError struct {
	Code    string `json:"code" validate:"required" example:"meter_already_exists"`
	Message string `json:"message,omitempty"`
}

// @description Possible codes: environment_inactive, meter_already_exists, meter_definition_conflict.
type MeterCreateConflict struct {
	Error MeterCreateConflictError `json:"error" validate:"required"`
}

type MeterCreateConflictError struct {
	Code    string `json:"code" validate:"required" example:"environment_inactive"`
	Message string `json:"message,omitempty"`
}

type MeterEnvironmentConflict struct {
	Error MeterEnvironmentConflictError `json:"error" validate:"required"`
}

type MeterEnvironmentConflictError struct {
	Code    string `json:"code" validate:"required" example:"environment_inactive"`
	Message string `json:"message,omitempty" example:"Activate Live before managing its meters."`
}

type MeterEnvironmentNotFound struct {
	Error MeterEnvironmentNotFoundError `json:"error" validate:"required"`
}

type MeterEnvironmentNotFoundError struct {
	Code    string `json:"code" validate:"required" example:"project_environment_not_found"`
	Message string `json:"message,omitempty" example:"This project environment is not available."`
}

type MeterFailed struct {
	Error MeterFailedError `json:"error" validate:"required"`
}

type MeterFailedError struct {
	Code    string `json:"code" validate:"required" example:"meter_operation_failed"`
	Message string `json:"message,omitempty" example:"Consumel could not complete the meter request. Try again shortly."`
}

// @description Possible codes: invalid_meter, invalid_request.
type MeterInvalid struct {
	Error MeterInvalidError `json:"error" validate:"required"`
}

type MeterInvalidError struct {
	Code    string `json:"code" validate:"required" example:"invalid_meter"`
	Message string `json:"message,omitempty"`
}

// @description Possible codes: invalid_cursor, invalid_request, invalid_search.
type MeterListInvalid struct {
	Error MeterListInvalidError `json:"error" validate:"required"`
}

type MeterListInvalidError struct {
	Code    string `json:"code" validate:"required" example:"invalid_cursor"`
	Message string `json:"message,omitempty"`
}

type MeterNotFound struct {
	Error MeterNotFoundError `json:"error" validate:"required"`
}

type MeterNotFoundError struct {
	Code    string `json:"code" validate:"required" example:"meter_not_found"`
	Message string `json:"message,omitempty" example:"This meter does not exist in the selected environment."`
}

// @description Possible codes: meter_not_found, project_environment_not_found.
type MeterOrEnvironmentNotFound struct {
	Error MeterOrEnvironmentNotFoundError `json:"error" validate:"required"`
}

type MeterOrEnvironmentNotFoundError struct {
	Code    string `json:"code" validate:"required" example:"meter_not_found"`
	Message string `json:"message,omitempty"`
}
