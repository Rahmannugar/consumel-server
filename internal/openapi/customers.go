package openapi

import (
	"time"
)

// @Summary List customers in the API key's project environment.
// @Tags Customers
// @Param q query string false "A case-insensitive customer ID, name, or email search." maxLength(120)
// @Param cursor query string false "The opaque next cursor from the previous page."
// @Param limit query int false "The number of customers to return." minimum(1) maximum(100) default(25)
// @Success 200 {object} Customers "Completed successfully."
// @Failure 400 {object} CustomerListInvalid "The list pagination or search input is invalid."
// @Failure 401 {object} InvalidAPIKey "The project API key is missing, malformed, revoked, replaced, or inactive."
// @Failure 500 {object} CustomerFailed "The customer request could not be completed."
// @Security projectAPIKey
// @Router /v1/customers [get]
func GetV1Customers() {}

// @Summary Create a customer in the API key's project environment.
// @Tags Customers
// @Param body body models.CreateCustomerRequest true "Customer to create."
// @Success 201 {object} Customer "Created."
// @Failure 400 {object} CustomerInvalid "The customer request or identifier is invalid."
// @Failure 401 {object} InvalidAPIKey "The project API key is missing, malformed, revoked, replaced, or inactive."
// @Failure 409 {object} CustomerConflict "The customer ID already exists in the selected environment."
// @Failure 500 {object} CustomerFailed "The customer request could not be completed."
// @Security projectAPIKey
// @Router /v1/customers [post]
func PostV1Customers() {}

// @Summary Return one customer from the API key's project environment.
// @Tags Customers
// @Param customerId path string true "The customer identifier supplied by the integrating application." maxLength(255)
// @Success 200 {object} Customer "Completed successfully."
// @Failure 400 {object} CustomerInvalid "The customer request or identifier is invalid."
// @Failure 401 {object} InvalidAPIKey "The project API key is missing, malformed, revoked, replaced, or inactive."
// @Failure 404 {object} CustomerNotFound "The customer is not present in the authenticated environment."
// @Failure 500 {object} CustomerFailed "The customer request could not be completed."
// @Security projectAPIKey
// @Router /v1/customers/{customerId} [get]
func GetV1CustomersCustomerId() {}

// @Summary Replace a customer's optional contact information and metadata.
// @Tags Customers
// @Param customerId path string true "The customer identifier supplied by the integrating application." maxLength(255)
// @Param body body models.UpdateCustomerRequest true "Replacement customer contact details and metadata."
// @Success 200 {object} Customer "Completed successfully."
// @Failure 400 {object} CustomerInvalid "The customer request or identifier is invalid."
// @Failure 401 {object} InvalidAPIKey "The project API key is missing, malformed, revoked, replaced, or inactive."
// @Failure 404 {object} CustomerNotFound "The customer is not present in the authenticated environment."
// @Failure 500 {object} CustomerFailed "The customer request could not be completed."
// @Security projectAPIKey
// @Router /v1/customers/{customerId} [put]
func PutV1CustomersCustomerId() {}

// @Summary List customers in the signed-in project workspace.
// @Tags Dashboard Customers
// @Param projectId path string true "The immutable ID of the project selected in the dashboard." format(uuid)
// @Param environment path string true "The selected isolated project environment." enums(sandbox, live)
// @Param q query string false "A case-insensitive customer ID, name, or email search." maxLength(120)
// @Param cursor query string false "The opaque next cursor from the previous page."
// @Param limit query int false "The number of customers to return." minimum(1) maximum(100) default(25)
// @Success 200 {object} Customers "Completed successfully."
// @Failure 400 {object} CustomerListInvalid "The list pagination or search input is invalid."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 404 {object} CustomerEnvironmentNotFound "The selected project environment is unavailable."
// @Failure 409 {object} CustomerEnvironmentConflict "The selected project environment is not active."
// @Failure 500 {object} CustomerFailed "The customer request could not be completed."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /v1/projects/{projectId}/environments/{environment}/customers [get]
func GetV1ProjectsProjectIdEnvironmentsEnvironmentCustomers() {}

// @Summary Create a customer from the signed-in project workspace.
// @Tags Dashboard Customers
// @Param projectId path string true "The immutable ID of the project selected in the dashboard." format(uuid)
// @Param environment path string true "The selected isolated project environment." enums(sandbox, live)
// @Param body body models.CreateCustomerRequest true "Customer to create."
// @Success 201 {object} Customer "Created."
// @Failure 400 {object} CustomerInvalid "The customer request or identifier is invalid."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 404 {object} CustomerEnvironmentNotFound "The selected project environment is unavailable."
// @Failure 409 {object} CustomerCreateConflict "The customer cannot be created in the selected environment."
// @Failure 500 {object} CustomerFailed "The customer request could not be completed."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /v1/projects/{projectId}/environments/{environment}/customers [post]
func PostV1ProjectsProjectIdEnvironmentsEnvironmentCustomers() {}

// @Summary Return one customer in the signed-in project workspace.
// @Tags Dashboard Customers
// @Param projectId path string true "The immutable ID of the project selected in the dashboard." format(uuid)
// @Param environment path string true "The selected isolated project environment." enums(sandbox, live)
// @Param customerId path string true "The customer identifier supplied by the integrating application." maxLength(255)
// @Success 200 {object} Customer "Completed successfully."
// @Failure 400 {object} CustomerInvalid "The customer request or identifier is invalid."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 404 {object} CustomerOrEnvironmentNotFound "The customer or selected project environment is unavailable."
// @Failure 409 {object} CustomerEnvironmentConflict "The selected project environment is not active."
// @Failure 500 {object} CustomerFailed "The customer request could not be completed."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /v1/projects/{projectId}/environments/{environment}/customers/{customerId} [get]
func GetV1ProjectsProjectIdEnvironmentsEnvironmentCustomersCustomerId() {}

// @Summary Replace a customer's optional contact information and supported metadata from the signed-in project workspace.
// @Tags Dashboard Customers
// @Param projectId path string true "The immutable ID of the project selected in the dashboard." format(uuid)
// @Param environment path string true "The selected isolated project environment." enums(sandbox, live)
// @Param customerId path string true "The customer identifier supplied by the integrating application." maxLength(255)
// @Param body body models.UpdateCustomerRequest true "Replacement customer contact details and metadata."
// @Success 200 {object} Customer "Completed successfully."
// @Failure 400 {object} CustomerInvalid "The customer request or identifier is invalid."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 404 {object} CustomerOrEnvironmentNotFound "The customer or selected project environment is unavailable."
// @Failure 409 {object} CustomerEnvironmentConflict "The selected project environment is not active."
// @Failure 500 {object} CustomerFailed "The customer request could not be completed."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /v1/projects/{projectId}/environments/{environment}/customers/{customerId} [put]
func PutV1ProjectsProjectIdEnvironmentsEnvironmentCustomersCustomerId() {}

type Customer struct {
	CreatedAt  time.Time        `json:"createdAt" validate:"required" example:"2026-09-28T12:00:00Z" format:"date-time"`
	CustomerID string           `json:"customerId" validate:"required,max=255" example:"user_123"`
	Email      *string          `json:"email" validate:"required,max=320" example:"jordan@example.com" format:"email"`
	Metadata   CustomerMetadata `json:"metadata" validate:"required"`
	Name       *string          `json:"name" validate:"required,max=200" example:"Jordan Lee"`
	UpdatedAt  time.Time        `json:"updatedAt" validate:"required" example:"2026-09-28T12:00:00Z" format:"date-time"`
}

type CustomerMetadata struct {
	Country  *string `json:"country,omitempty" example:"US" pattern:"^[A-Z]{2}$"`
	Location *string `json:"location,omitempty" validate:"max=120" example:"New York, NY"`
	Plan     *string `json:"plan,omitempty" validate:"max=120" example:"growth"`
}

type Customers struct {
	Customers  []Customer `json:"customers" validate:"required"`
	NextCursor *string    `json:"nextCursor" validate:"required"`
}

type CustomerConflict struct {
	Error CustomerConflictError `json:"error" validate:"required"`
}

type CustomerConflictError struct {
	Code    string `json:"code" validate:"required" example:"customer_already_exists"`
	Message string `json:"message,omitempty" example:"This customer ID already exists in the selected environment."`
}

// @description Possible codes: customer_already_exists, environment_inactive.
type CustomerCreateConflict struct {
	Error CustomerCreateConflictError `json:"error" validate:"required"`
}

type CustomerCreateConflictError struct {
	Code    string `json:"code" validate:"required" example:"customer_already_exists"`
	Message string `json:"message,omitempty"`
}

type CustomerEnvironmentConflict struct {
	Error CustomerEnvironmentConflictError `json:"error" validate:"required"`
}

type CustomerEnvironmentConflictError struct {
	Code    string `json:"code" validate:"required" example:"environment_inactive"`
	Message string `json:"message,omitempty" example:"Activate Live before managing its customers."`
}

type CustomerEnvironmentNotFound struct {
	Error CustomerEnvironmentNotFoundError `json:"error" validate:"required"`
}

type CustomerEnvironmentNotFoundError struct {
	Code    string `json:"code" validate:"required" example:"project_environment_not_found"`
	Message string `json:"message,omitempty" example:"This project environment is not available."`
}

type CustomerFailed struct {
	Error CustomerFailedError `json:"error" validate:"required"`
}

type CustomerFailedError struct {
	Code    string `json:"code" validate:"required" example:"customer_operation_failed"`
	Message string `json:"message,omitempty" example:"Consumel could not complete the customer request. Try again shortly."`
}

// @description Possible codes: invalid_customer, invalid_request.
type CustomerInvalid struct {
	Error CustomerInvalidError `json:"error" validate:"required"`
}

type CustomerInvalidError struct {
	Code    string `json:"code" validate:"required" example:"invalid_customer"`
	Message string `json:"message,omitempty"`
}

// @description Possible codes: invalid_cursor, invalid_request, invalid_search.
type CustomerListInvalid struct {
	Error CustomerListInvalidError `json:"error" validate:"required"`
}

type CustomerListInvalidError struct {
	Code    string `json:"code" validate:"required" example:"invalid_cursor"`
	Message string `json:"message,omitempty"`
}

type CustomerNotFound struct {
	Error CustomerNotFoundError `json:"error" validate:"required"`
}

type CustomerNotFoundError struct {
	Code    string `json:"code" validate:"required" example:"customer_not_found"`
	Message string `json:"message,omitempty" example:"This customer does not exist in the selected environment."`
}

// @description Possible codes: customer_not_found, project_environment_not_found.
type CustomerOrEnvironmentNotFound struct {
	Error CustomerOrEnvironmentNotFoundError `json:"error" validate:"required"`
}

type CustomerOrEnvironmentNotFoundError struct {
	Code    string `json:"code" validate:"required" example:"customer_not_found"`
	Message string `json:"message,omitempty"`
}
