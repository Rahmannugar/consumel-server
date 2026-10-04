package openapi

import (
	"github.com/google/uuid"
	"time"
)

// @Summary Add an optionally expiring, idempotent quantity to a customer and meter balance.
// @Tags Balances
// @Param Idempotency-Key header string true "A UUID v7 that identifies this logical balance addition. Reuse it only when retrying the same request." format(uuid)
// @Param body body models.AddBalanceRequest true "Balance quantity to add."
// @Success 200 {object} Balance "Completed successfully."
// @Header 200 {string} Idempotency-Replayed "Present with the value true when Consumel returns the original result of a committed retry."
// @Failure 400 {object} BalanceInvalid "The balance request, resource key, or idempotency key is invalid."
// @Failure 401 {object} InvalidAPIKey "The project API key is missing, malformed, revoked, replaced, or inactive."
// @Failure 404 {object} BalanceSubjectNotFound "The customer or active meter is unavailable."
// @Failure 409 {object} BalanceConflict "The balance addition conflicts with prior state."
// @Failure 429 {object} RateLimited "Too many attempts were made."
// @Failure 500 {object} BalanceFailed "The balance request could not be completed."
// @Security projectAPIKey
// @Router /v1/balances [post]
func PostV1Balances() {}

// @Summary Set a customer and meter balance to an exact quantity.
// @Tags Balances
// @Param customerId path string true "The customer identifier supplied by the integrating application." maxLength(255)
// @Param meterKey path string true "The stable meter key supplied when the meter was created. Must match `^[a-z][a-z0-9_-]*$`." maxLength(120)
// @Param body body models.SetBalanceRequest true "Exact balance quantity to set."
// @Success 200 {object} Balance "Completed successfully."
// @Failure 400 {object} BalanceInvalid "The balance request, resource key, or idempotency key is invalid."
// @Failure 401 {object} InvalidAPIKey "The project API key is missing, malformed, revoked, replaced, or inactive."
// @Failure 404 {object} BalanceSubjectNotFound "The customer or active meter is unavailable."
// @Failure 429 {object} RateLimited "Too many attempts were made."
// @Failure 500 {object} BalanceFailed "The balance request could not be completed."
// @Security projectAPIKey
// @Router /v1/balances/{customerId}/{meterKey} [put]
func PutV1BalancesCustomerIdMeterKey() {}

// @Summary Atomically record usage and apply the active meter's balance behavior.
// @Tags Consumption
// @Param Idempotency-Key header string true "A UUID v7 that identifies this logical consume operation. Reuse it only when retrying the same request." format(uuid)
// @Param body body models.ConsumeRequest true "Usage to record."
// @Success 200 {object} UsageEvent "Completed successfully."
// @Header 200 {string} Idempotency-Replayed "Present with the value true when Consumel returns the original result of a committed retry."
// @Failure 400 {object} ConsumeInvalid "The consume request or idempotency key is invalid."
// @Failure 401 {object} InvalidAPIKey "The project API key is missing, malformed, revoked, replaced, or inactive."
// @Failure 404 {object} ConsumeMeterNotFound "The active meter is unavailable."
// @Failure 409 {object} ConsumeConflict "The consume operation conflicts with the current balance or prior idempotent request."
// @Failure 429 {object} RateLimited "Too many attempts were made."
// @Failure 500 {object} ConsumeFailed "The consume operation could not be completed."
// @Security projectAPIKey
// @Router /v1/consume [post]
func PostV1Consume() {}

// @Summary List a customer's active meter balances in the API key's project environment.
// @Tags Balances
// @Param customerId path string true "The customer identifier supplied by the integrating application." maxLength(255)
// @Success 200 {object} Balances "Completed successfully."
// @Failure 400 {object} BalanceInvalid "The balance request, resource key, or idempotency key is invalid."
// @Failure 401 {object} InvalidAPIKey "The project API key is missing, malformed, revoked, replaced, or inactive."
// @Failure 404 {object} BalanceNotFound "The customer, active meter, or balance is unavailable."
// @Failure 500 {object} BalanceFailed "The balance request could not be completed."
// @Security projectAPIKey
// @Router /v1/customers/{customerId}/balances [get]
func GetV1CustomersCustomerIdBalances() {}

// @Summary Return one customer and meter balance from the API key's project environment.
// @Tags Balances
// @Param customerId path string true "The customer identifier supplied by the integrating application." maxLength(255)
// @Param meterKey path string true "The stable meter key supplied when the meter was created. Must match `^[a-z][a-z0-9_-]*$`." maxLength(120)
// @Success 200 {object} Balance "Completed successfully."
// @Failure 400 {object} BalanceInvalid "The balance request, resource key, or idempotency key is invalid."
// @Failure 401 {object} InvalidAPIKey "The project API key is missing, malformed, revoked, replaced, or inactive."
// @Failure 404 {object} BalanceNotFound "The customer, active meter, or balance is unavailable."
// @Failure 500 {object} BalanceFailed "The balance request could not be completed."
// @Security projectAPIKey
// @Router /v1/customers/{customerId}/balances/{meterKey} [get]
func GetV1CustomersCustomerIdBalancesMeterKey() {}

// @Summary List the entitlement grants that compose one customer and meter balance.
// @Tags Balances
// @Param customerId path string true "The customer identifier supplied by the integrating application." maxLength(255)
// @Param meterKey path string true "The stable meter key supplied when the meter was created. Must match `^[a-z][a-z0-9_-]*$`." maxLength(120)
// @Param cursor query string false "The opaque next cursor from the previous page."
// @Param limit query int false "The number of records to return." minimum(1) maximum(100) default(25)
// @Success 200 {object} EntitlementGrants "Completed successfully."
// @Failure 400 {object} BalanceInvalid "The balance request, resource key, or idempotency key is invalid."
// @Failure 401 {object} InvalidAPIKey "The project API key is missing, malformed, revoked, replaced, or inactive."
// @Failure 404 {object} BalanceNotFound "The customer, active meter, or balance is unavailable."
// @Failure 500 {object} BalanceFailed "The balance request could not be completed."
// @Security projectAPIKey
// @Router /v1/customers/{customerId}/balances/{meterKey}/grants [get]
func GetV1CustomersCustomerIdBalancesMeterKeyGrants() {}

// @Summary List adjustments, usage debits, and expirations for one customer and meter balance.
// @Tags Balances
// @Param customerId path string true "The customer identifier supplied by the integrating application." maxLength(255)
// @Param meterKey path string true "The stable meter key supplied when the meter was created. Must match `^[a-z][a-z0-9_-]*$`." maxLength(120)
// @Param cursor query string false "The opaque next cursor from the previous page."
// @Param limit query int false "The number of records to return." minimum(1) maximum(100) default(25)
// @Success 200 {object} BalanceActivityList "Completed successfully."
// @Failure 400 {object} BalanceInvalid "The balance request, resource key, or idempotency key is invalid."
// @Failure 401 {object} InvalidAPIKey "The project API key is missing, malformed, revoked, replaced, or inactive."
// @Failure 404 {object} BalanceNotFound "The customer, active meter, or balance is unavailable."
// @Failure 500 {object} BalanceFailed "The balance request could not be completed."
// @Security projectAPIKey
// @Router /v1/customers/{customerId}/balances/{meterKey}/history [get]
func GetV1CustomersCustomerIdBalancesMeterKeyHistory() {}

// @Summary Add an optionally expiring, idempotent quantity to a customer and meter balance from the signed-in project workspace.
// @Tags Dashboard Balances
// @Param projectId path string true "The immutable ID of the project selected in the dashboard." format(uuid)
// @Param environment path string true "The selected isolated project environment." enums(sandbox, live)
// @Param Idempotency-Key header string true "A UUID v7 that identifies this logical balance addition. Reuse it only when retrying the same request." format(uuid)
// @Param body body models.AddBalanceRequest true "Balance quantity to add."
// @Success 200 {object} Balance "Completed successfully."
// @Header 200 {string} Idempotency-Replayed "Present with the value true when Consumel returns the original result of a committed retry."
// @Failure 400 {object} BalanceInvalid "The balance request, resource key, or idempotency key is invalid."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 404 {object} BalanceSubjectOrEnvironmentNotFound "The customer, active meter, or selected project environment is unavailable."
// @Failure 409 {object} BalanceOrEnvironmentConflict "The balance addition conflicts with prior state or the selected environment is inactive."
// @Failure 500 {object} BalanceFailed "The balance request could not be completed."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /v1/projects/{projectId}/environments/{environment}/balances [post]
func PostV1ProjectsProjectIdEnvironmentsEnvironmentBalances() {}

// @Summary Set a customer and meter balance to an exact quantity from the signed-in project workspace.
// @Tags Dashboard Balances
// @Param projectId path string true "The immutable ID of the project selected in the dashboard." format(uuid)
// @Param environment path string true "The selected isolated project environment." enums(sandbox, live)
// @Param customerId path string true "The customer identifier supplied by the integrating application." maxLength(255)
// @Param meterKey path string true "The stable meter key supplied when the meter was created. Must match `^[a-z][a-z0-9_-]*$`." maxLength(120)
// @Param body body models.SetBalanceRequest true "Exact balance quantity to set."
// @Success 200 {object} Balance "Completed successfully."
// @Failure 400 {object} BalanceInvalid "The balance request, resource key, or idempotency key is invalid."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 404 {object} BalanceSubjectOrEnvironmentNotFound "The customer, active meter, or selected project environment is unavailable."
// @Failure 409 {object} BalanceEnvironmentConflict "The selected project environment is not active."
// @Failure 500 {object} BalanceFailed "The balance request could not be completed."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /v1/projects/{projectId}/environments/{environment}/balances/{customerId}/{meterKey} [put]
func PutV1ProjectsProjectIdEnvironmentsEnvironmentBalancesCustomerIdMeterKey() {}

// @Summary List a customer's active meter balances from the signed-in project workspace.
// @Tags Dashboard Balances
// @Param projectId path string true "The immutable ID of the project selected in the dashboard." format(uuid)
// @Param environment path string true "The selected isolated project environment." enums(sandbox, live)
// @Param customerId path string true "The customer identifier supplied by the integrating application." maxLength(255)
// @Success 200 {object} Balances "Completed successfully."
// @Failure 400 {object} BalanceInvalid "The balance request, resource key, or idempotency key is invalid."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 404 {object} BalanceOrEnvironmentNotFound "The customer, active meter, balance, or selected environment is unavailable."
// @Failure 409 {object} BalanceEnvironmentConflict "The selected project environment is not active."
// @Failure 500 {object} BalanceFailed "The balance request could not be completed."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /v1/projects/{projectId}/environments/{environment}/customers/{customerId}/balances [get]
func GetV1ProjectsProjectIdEnvironmentsEnvironmentCustomersCustomerIdBalances() {}

// @Summary Return one customer and meter balance from the signed-in project workspace.
// @Tags Dashboard Balances
// @Param projectId path string true "The immutable ID of the project selected in the dashboard." format(uuid)
// @Param environment path string true "The selected isolated project environment." enums(sandbox, live)
// @Param customerId path string true "The customer identifier supplied by the integrating application." maxLength(255)
// @Param meterKey path string true "The stable meter key supplied when the meter was created. Must match `^[a-z][a-z0-9_-]*$`." maxLength(120)
// @Success 200 {object} Balance "Completed successfully."
// @Failure 400 {object} BalanceInvalid "The balance request, resource key, or idempotency key is invalid."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 404 {object} BalanceOrEnvironmentNotFound "The customer, active meter, balance, or selected environment is unavailable."
// @Failure 409 {object} BalanceEnvironmentConflict "The selected project environment is not active."
// @Failure 500 {object} BalanceFailed "The balance request could not be completed."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /v1/projects/{projectId}/environments/{environment}/customers/{customerId}/balances/{meterKey} [get]
func GetV1ProjectsProjectIdEnvironmentsEnvironmentCustomersCustomerIdBalancesMeterKey() {}

// @Summary List the entitlement grants that compose one customer and meter balance in the signed-in project workspace.
// @Tags Dashboard Balances
// @Param projectId path string true "The immutable ID of the project selected in the dashboard." format(uuid)
// @Param environment path string true "The selected isolated project environment." enums(sandbox, live)
// @Param customerId path string true "The customer identifier supplied by the integrating application." maxLength(255)
// @Param meterKey path string true "The stable meter key supplied when the meter was created. Must match `^[a-z][a-z0-9_-]*$`." maxLength(120)
// @Param cursor query string false "The opaque next cursor from the previous page."
// @Param limit query int false "The number of records to return." minimum(1) maximum(100) default(25)
// @Success 200 {object} EntitlementGrants "Completed successfully."
// @Failure 400 {object} BalanceInvalid "The balance request, resource key, or idempotency key is invalid."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 404 {object} BalanceOrEnvironmentNotFound "The customer, active meter, balance, or selected environment is unavailable."
// @Failure 409 {object} BalanceEnvironmentConflict "The selected project environment is not active."
// @Failure 500 {object} BalanceFailed "The balance request could not be completed."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /v1/projects/{projectId}/environments/{environment}/customers/{customerId}/balances/{meterKey}/grants [get]
func GetV1ProjectsProjectIdEnvironmentsEnvironmentCustomersCustomerIdBalancesMeterKeyGrants() {}

// @Summary List adjustments, usage debits, and expirations for one customer and meter balance in the signed-in project workspace.
// @Tags Dashboard Balances
// @Param projectId path string true "The immutable ID of the project selected in the dashboard." format(uuid)
// @Param environment path string true "The selected isolated project environment." enums(sandbox, live)
// @Param customerId path string true "The customer identifier supplied by the integrating application." maxLength(255)
// @Param meterKey path string true "The stable meter key supplied when the meter was created. Must match `^[a-z][a-z0-9_-]*$`." maxLength(120)
// @Param cursor query string false "The opaque next cursor from the previous page."
// @Param limit query int false "The number of records to return." minimum(1) maximum(100) default(25)
// @Success 200 {object} BalanceActivityList "Completed successfully."
// @Failure 400 {object} BalanceInvalid "The balance request, resource key, or idempotency key is invalid."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 404 {object} BalanceOrEnvironmentNotFound "The customer, active meter, balance, or selected environment is unavailable."
// @Failure 409 {object} BalanceEnvironmentConflict "The selected project environment is not active."
// @Failure 500 {object} BalanceFailed "The balance request could not be completed."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /v1/projects/{projectId}/environments/{environment}/customers/{customerId}/balances/{meterKey}/history [get]
func GetV1ProjectsProjectIdEnvironmentsEnvironmentCustomersCustomerIdBalancesMeterKeyHistory() {}

// @Summary List accepted and denied usage operations in the signed-in project workspace.
// @Tags Dashboard Events
// @Param projectId path string true "The immutable ID of the project selected in the dashboard." format(uuid)
// @Param environment path string true "The selected isolated project environment." enums(sandbox, live)
// @Param status query string false "Filter by the persisted operation outcome." enums(accepted, denied)
// @Param customerId query string false "Return usage operations for this exact customer identifier." maxLength(255)
// @Param meterKey query string false "Return usage operations for this exact meter key. Must match `^[a-z][a-z0-9_-]*$`." maxLength(120)
// @Param from query string false "Inclusive RFC 3339 start time. Supply from and to together; the range may span at most one year." format(date-time)
// @Param to query string false "Exclusive RFC 3339 end time. Supply from and to together." format(date-time)
// @Param cursor query string false "The opaque next cursor from the previous page."
// @Param limit query int false "The number of events to return." minimum(1) maximum(100) default(50)
// @Success 200 {object} UsageOperations "Completed successfully."
// @Failure 400 {object} OperationListInvalid "The event list filter or pagination input is invalid."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 404 {object} OperationEnvironmentNotFound "The selected project environment is unavailable."
// @Failure 409 {object} OperationEnvironmentConflict "The selected project environment is not active."
// @Failure 500 {object} OperationListFailed "The event list could not be loaded."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /v1/projects/{projectId}/environments/{environment}/events [get]
func GetV1ProjectsProjectIdEnvironmentsEnvironmentEvents() {}

// @Summary Stream newly persisted usage operations to the signed-in project workspace.
// @Tags Dashboard Events
// @Param projectId path string true "The immutable ID of the project selected in the dashboard." format(uuid)
// @Param environment path string true "The selected isolated project environment." enums(sandbox, live)
// @Param cursor query string false "The last Redis stream cursor received by the client. Browsers normally reconnect through Last-Event-ID automatically. Must match `^[0-9]+-[0-9]+$`."
// @Success 200 {string} string "A server-sent event stream. Each usage.operation event contains a UsageOperation JSON payload and a Redis stream cursor used for reconnection."
// @Failure 400 {object} OperationListInvalid "The event list filter or pagination input is invalid."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 404 {object} OperationEnvironmentNotFound "The selected project environment is unavailable."
// @Failure 409 {object} OperationEnvironmentConflict "The selected project environment is not active."
// @Failure 500 {object} OperationListFailed "The event list could not be loaded."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /v1/projects/{projectId}/environments/{environment}/events/stream [get]
func GetV1ProjectsProjectIdEnvironmentsEnvironmentEventsStream() {}

// @Summary Return bounded usage analytics for the API key's project environment.
// @Tags Analytics
// @Param from query string true "Inclusive RFC 3339 start time." format(date-time)
// @Param to query string true "Exclusive RFC 3339 end time." format(date-time)
// @Param interval query string true "UTC aggregation interval. Hourly ranges may span 31 days; daily ranges may span one year." enums(hour, day)
// @Param customerId query string false "Aggregate one exact customer identifier." maxLength(255)
// @Param meterKey query string false "Aggregate one exact meter key. Must match `^[a-z][a-z0-9_-]*$`." maxLength(120)
// @Success 200 {object} UsageAnalytics "Completed successfully. Missing UTC buckets are returned with zero values."
// @Failure 400 {object} AnalyticsInvalid "The analytics filters are invalid."
// @Failure 401 {object} InvalidAPIKey "The project API key is missing, malformed, revoked, replaced, or inactive."
// @Failure 500 {object} AnalyticsFailed "Analytics could not be loaded."
// @Security projectAPIKey
// @Router /v1/analytics [get]
func GetV1Analytics() {}

// @Summary Return bounded usage analytics from the signed-in project workspace.
// @Tags Dashboard Analytics
// @Param projectId path string true "The immutable ID of the project selected in the dashboard." format(uuid)
// @Param environment path string true "The selected isolated project environment." enums(sandbox, live)
// @Param from query string true "Inclusive RFC 3339 start time." format(date-time)
// @Param to query string true "Exclusive RFC 3339 end time." format(date-time)
// @Param interval query string true "UTC aggregation interval. Hourly ranges may span 31 days; daily ranges may span one year." enums(hour, day)
// @Param customerId query string false "Aggregate one exact customer identifier." maxLength(255)
// @Param meterKey query string false "Aggregate one exact meter key. Must match `^[a-z][a-z0-9_-]*$`." maxLength(120)
// @Success 200 {object} UsageAnalytics "Completed successfully. Missing UTC buckets are returned with zero values."
// @Failure 400 {object} AnalyticsInvalid "The analytics filters are invalid."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 404 {object} OperationEnvironmentNotFound "The selected project environment is unavailable."
// @Failure 409 {object} OperationEnvironmentConflict "The selected project environment is not active."
// @Failure 500 {object} AnalyticsFailed "Analytics could not be loaded."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /v1/projects/{projectId}/environments/{environment}/analytics [get]
func GetV1ProjectsProjectIdEnvironmentsEnvironmentAnalytics() {}

type Balance struct {
	CreatedAt     time.Time  `json:"createdAt" validate:"required" example:"2026-09-28T12:00:00Z" format:"date-time"`
	CustomerID    string     `json:"customerId" validate:"required,max=255" example:"user_123"`
	ID            uuid.UUID  `json:"id" validate:"required" example:"0199aa81-ce8c-73bf-a880-8e84654b9a6c" format:"uuid"`
	MeterKey      string     `json:"meterKey" validate:"required,max=120" example:"api_calls" pattern:"^[a-z][a-z0-9_-]*$"`
	NextExpiresAt *time.Time `json:"nextExpiresAt" validate:"required" example:"2026-10-31T00:00:00Z" format:"date-time"`
	Quantity      int64      `json:"quantity" validate:"required,min=0" example:"10000" format:"int64"`
	UpdatedAt     time.Time  `json:"updatedAt" validate:"required" example:"2026-09-28T12:00:00Z" format:"date-time"`
}

type BalanceActivity struct {
	ExpiresAt         *time.Time `json:"expiresAt" validate:"required" format:"date-time"`
	ID                uuid.UUID  `json:"id" validate:"required" example:"0199aa81-ce8c-73bf-a880-8e84654b9a71" format:"uuid"`
	Kind              string     `json:"kind" validate:"required" example:"usage" enums:"add,set,usage,expiration"`
	OccurredAt        time.Time  `json:"occurredAt" validate:"required" example:"2026-09-29T12:00:00Z" format:"date-time"`
	QuantityChange    int64      `json:"quantityChange" validate:"required" example:"-500" format:"int64"`
	ResultingQuantity *int64     `json:"resultingQuantity" validate:"required,min=0" example:"7000" format:"int64"`
	SourceType        string     `json:"sourceType" validate:"required" example:"api_key"`
}

type BalanceActivityList struct {
	Activity   []BalanceActivity `json:"activity" validate:"required"`
	NextCursor *string           `json:"nextCursor" validate:"required"`
}

type Balances struct {
	Balances []Balance `json:"balances" validate:"required"`
}

type EntitlementGrant struct {
	CreatedAt         time.Time  `json:"createdAt" validate:"required" example:"2026-09-28T12:00:00Z" format:"date-time"`
	ExpiresAt         *time.Time `json:"expiresAt" validate:"required" example:"2026-10-31T00:00:00Z" format:"date-time"`
	GrantedQuantity   int64      `json:"grantedQuantity" validate:"required,min=1" example:"10000" format:"int64"`
	ID                uuid.UUID  `json:"id" validate:"required" example:"0199aa81-ce8c-73bf-a880-8e84654b9a70" format:"uuid"`
	RemainingQuantity int64      `json:"remainingQuantity" validate:"required,min=0" example:"7500" format:"int64"`
	Status            string     `json:"status" validate:"required" example:"active" enums:"active,exhausted,expired"`
}

type EntitlementGrants struct {
	Grants     []EntitlementGrant `json:"grants" validate:"required"`
	NextCursor *string            `json:"nextCursor" validate:"required"`
}

type UsageEvent struct {
	BalanceDebited   int64     `json:"balanceDebited" validate:"required,min=0" example:"500" format:"int64"`
	Billable         bool      `json:"billable" validate:"required" example:"true"`
	CreatedAt        time.Time `json:"createdAt" validate:"required" example:"2026-09-29T12:00:00Z" format:"date-time"`
	CustomerID       string    `json:"customerId" validate:"required,max=255" example:"customer_123"`
	ID               uuid.UUID `json:"id" validate:"required" example:"0199aad1-f00d-7ae0-935f-cde0bd9f3ba5" format:"uuid"`
	MeterKey         string    `json:"meterKey" validate:"required,max=120" example:"api_calls" pattern:"^[a-z][a-z0-9_-]*$"`
	MeterType        string    `json:"meterType" validate:"required" example:"prepaid" enums:"prepaid,postpaid,hybrid"`
	Quantity         int64     `json:"quantity" validate:"required,min=1" example:"500" format:"int64"`
	RemainingBalance *int64    `json:"remainingBalance" validate:"required,min=0" example:"9500" format:"int64"`
}

type UsageOperation struct {
	BalanceDebited   int64      `json:"balanceDebited" validate:"required,min=0" example:"500" format:"int64"`
	Billable         bool       `json:"billable" validate:"required" example:"true"`
	CreatedAt        time.Time  `json:"createdAt" validate:"required" example:"2026-09-29T12:00:00Z" format:"date-time"`
	CustomerID       string     `json:"customerId" validate:"required" example:"customer_123"`
	DenialReason     *string    `json:"denialReason" validate:"required"`
	ID               uuid.UUID  `json:"id" validate:"required" example:"0199aad1-f00d-7ae0-935f-cde0bd9f3ba5" format:"uuid"`
	LastReplayedAt   *time.Time `json:"lastReplayedAt" validate:"required" example:"2026-09-29T12:01:00Z" format:"date-time"`
	MeterKey         string     `json:"meterKey" validate:"required" example:"api_calls"`
	MeterType        string     `json:"meterType" validate:"required" example:"prepaid" enums:"prepaid,postpaid,hybrid"`
	Quantity         int64      `json:"quantity" validate:"required,min=1" example:"500" format:"int64"`
	RemainingBalance *int64     `json:"remainingBalance" validate:"required,min=0" example:"9500" format:"int64"`
	ReplayCount      int64      `json:"replayCount" validate:"required,min=0" example:"1" format:"int64"`
	Status           string     `json:"status" validate:"required" example:"accepted" enums:"accepted,denied"`
}

type UsageOperations struct {
	NextCursor *string          `json:"nextCursor" validate:"required"`
	Operations []UsageOperation `json:"operations" validate:"required"`
}

type UsageAnalytics struct {
	Buckets  []UsageAnalyticsBucket `json:"buckets" validate:"required"`
	From     time.Time              `json:"from" validate:"required" example:"2026-09-01T00:00:00Z" format:"date-time"`
	Interval string                 `json:"interval" validate:"required" example:"day" enums:"hour,day"`
	Summary  UsageAnalyticsSummary  `json:"summary" validate:"required"`
	To       time.Time              `json:"to" validate:"required" example:"2026-10-01T00:00:00Z" format:"date-time"`
}

type UsageAnalyticsSummary struct {
	AcceptedOperations int64 `json:"acceptedOperations" validate:"required,min=0" example:"950" format:"int64"`
	AcceptedQuantity   int64 `json:"acceptedQuantity" validate:"required,min=0" example:"128400" format:"int64"`
	BillableOperations int64 `json:"billableOperations" validate:"required,min=0" example:"720" format:"int64"`
	DeniedOperations   int64 `json:"deniedOperations" validate:"required,min=0" example:"12" format:"int64"`
	DeniedQuantity     int64 `json:"deniedQuantity" validate:"required,min=0" example:"1800" format:"int64"`
}

type UsageAnalyticsBucket struct {
	AcceptedOperations int64     `json:"acceptedOperations" validate:"required,min=0" example:"42" format:"int64"`
	AcceptedQuantity   int64     `json:"acceptedQuantity" validate:"required,min=0" example:"6200" format:"int64"`
	BillableOperations int64     `json:"billableOperations" validate:"required,min=0" example:"35" format:"int64"`
	DeniedOperations   int64     `json:"deniedOperations" validate:"required,min=0" example:"2" format:"int64"`
	DeniedQuantity     int64     `json:"deniedQuantity" validate:"required,min=0" example:"250" format:"int64"`
	Start              time.Time `json:"start" validate:"required" example:"2026-09-28T00:00:00Z" format:"date-time"`
}

type AnalyticsInvalid struct {
	Error AnalyticsInvalidError `json:"error" validate:"required"`
}

type AnalyticsInvalidError struct {
	Code    string `json:"code" validate:"required" example:"invalid_filter"`
	Message string `json:"message,omitempty" example:"Choose valid customer, meter, interval, and date filters. Hourly ranges may span 31 days; daily ranges may span one year."`
}

type AnalyticsFailed struct {
	Error AnalyticsFailedError `json:"error" validate:"required"`
}

type AnalyticsFailedError struct {
	Code    string `json:"code" validate:"required" example:"analytics_failed"`
	Message string `json:"message,omitempty" example:"Consumel could not load analytics. Try again shortly."`
}

// @description Possible codes: balance_limit_exceeded, idempotency_key_conflict.
type BalanceConflict struct {
	Error BalanceConflictError `json:"error" validate:"required"`
}

type BalanceConflictError struct {
	Code    string `json:"code" validate:"required" example:"balance_limit_exceeded"`
	Message string `json:"message,omitempty"`
}

type BalanceEnvironmentConflict struct {
	Error BalanceEnvironmentConflictError `json:"error" validate:"required"`
}

type BalanceEnvironmentConflictError struct {
	Code    string `json:"code" validate:"required" example:"environment_inactive"`
	Message string `json:"message,omitempty" example:"Activate Live before managing its balances."`
}

type BalanceFailed struct {
	Error BalanceFailedError `json:"error" validate:"required"`
}

type BalanceFailedError struct {
	Code    string `json:"code" validate:"required" example:"balance_operation_failed"`
	Message string `json:"message,omitempty" example:"Consumel could not complete the balance request. Try again shortly."`
}

// @description Possible codes: invalid_balance, invalid_request.
type BalanceInvalid struct {
	Error BalanceInvalidError `json:"error" validate:"required"`
}

type BalanceInvalidError struct {
	Code    string `json:"code" validate:"required" example:"invalid_balance"`
	Message string `json:"message,omitempty"`
}

// @description Possible codes: balance_not_found, balance_subject_not_found.
type BalanceNotFound struct {
	Error BalanceNotFoundError `json:"error" validate:"required"`
}

type BalanceNotFoundError struct {
	Code    string `json:"code" validate:"required" example:"balance_not_found"`
	Message string `json:"message,omitempty"`
}

// @description Possible codes: balance_limit_exceeded, environment_inactive, idempotency_key_conflict.
type BalanceOrEnvironmentConflict struct {
	Error BalanceOrEnvironmentConflictError `json:"error" validate:"required"`
}

type BalanceOrEnvironmentConflictError struct {
	Code    string `json:"code" validate:"required" example:"balance_limit_exceeded"`
	Message string `json:"message,omitempty"`
}

// @description Possible codes: balance_not_found, balance_subject_not_found, project_environment_not_found.
type BalanceOrEnvironmentNotFound struct {
	Error BalanceOrEnvironmentNotFoundError `json:"error" validate:"required"`
}

type BalanceOrEnvironmentNotFoundError struct {
	Code    string `json:"code" validate:"required" example:"balance_not_found"`
	Message string `json:"message,omitempty"`
}

type BalanceSubjectNotFound struct {
	Error BalanceSubjectNotFoundError `json:"error" validate:"required"`
}

type BalanceSubjectNotFoundError struct {
	Code    string `json:"code" validate:"required" example:"balance_subject_not_found"`
	Message string `json:"message,omitempty" example:"The customer or active meter does not exist in this environment."`
}

// @description Possible codes: balance_subject_not_found, project_environment_not_found.
type BalanceSubjectOrEnvironmentNotFound struct {
	Error BalanceSubjectOrEnvironmentNotFoundError `json:"error" validate:"required"`
}

type BalanceSubjectOrEnvironmentNotFoundError struct {
	Code    string `json:"code" validate:"required" example:"balance_subject_not_found"`
	Message string `json:"message,omitempty"`
}

// @description Possible codes: idempotency_key_conflict, insufficient_balance.
type ConsumeConflict struct {
	Error ConsumeConflictError `json:"error" validate:"required"`
}

type ConsumeConflictError struct {
	Code    string `json:"code" validate:"required" example:"idempotency_key_conflict"`
	Message string `json:"message,omitempty"`
}

type ConsumeFailed struct {
	Error ConsumeFailedError `json:"error" validate:"required"`
}

type ConsumeFailedError struct {
	Code    string `json:"code" validate:"required" example:"consume_failed"`
	Message string `json:"message,omitempty" example:"Consumel could not process this usage. Try again shortly."`
}

type ConsumeInvalid struct {
	Error ConsumeInvalidError `json:"error" validate:"required"`
}

type ConsumeInvalidError struct {
	Code    string `json:"code" validate:"required" example:"invalid_consume"`
	Message string `json:"message,omitempty" example:"Check the customer ID, meter key, quantity, and idempotency key."`
}

type ConsumeMeterNotFound struct {
	Error ConsumeMeterNotFoundError `json:"error" validate:"required"`
}

type ConsumeMeterNotFoundError struct {
	Code    string `json:"code" validate:"required" example:"meter_not_found"`
	Message string `json:"message,omitempty" example:"The active meter does not exist in this environment."`
}

type OperationEnvironmentConflict struct {
	Error OperationEnvironmentConflictError `json:"error" validate:"required"`
}

type OperationEnvironmentConflictError struct {
	Code    string `json:"code" validate:"required" example:"environment_inactive"`
	Message string `json:"message,omitempty" example:"Activate Live before viewing its events."`
}

type OperationEnvironmentNotFound struct {
	Error OperationEnvironmentNotFoundError `json:"error" validate:"required"`
}

type OperationEnvironmentNotFoundError struct {
	Code    string `json:"code" validate:"required" example:"project_environment_not_found"`
	Message string `json:"message,omitempty" example:"This project environment is not available."`
}

type OperationListFailed struct {
	Error OperationListFailedError `json:"error" validate:"required"`
}

type OperationListFailedError struct {
	Code    string `json:"code" validate:"required" example:"operation_list_failed"`
	Message string `json:"message,omitempty" example:"Consumel could not load events. Try again shortly."`
}

// @description Possible codes: invalid_cursor, invalid_filter, invalid_request.
type OperationListInvalid struct {
	Error OperationListInvalidError `json:"error" validate:"required"`
}

type OperationListInvalidError struct {
	Code    string `json:"code" validate:"required" example:"invalid_cursor"`
	Message string `json:"message,omitempty"`
}
