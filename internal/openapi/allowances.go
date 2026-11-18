package openapi

import (
	"time"

	"github.com/google/uuid"
)

type RecurringAllowanceRevision struct {
	Version     int64     `json:"version" validate:"required" example:"1" format:"int64"`
	EffectiveAt time.Time `json:"effectiveAt" validate:"required" example:"2026-12-01T00:00:00Z" format:"date-time"`
	Quantity    int64     `json:"quantity" validate:"required" example:"1000" format:"int64"`
	RolloverCap int64     `json:"rolloverCap" validate:"required" example:"2000" format:"int64"`
	Enabled     bool      `json:"enabled" validate:"required" example:"true"`
}
type RecurringAllowance struct {
	ID           uuid.UUID                   `json:"id" validate:"required" example:"019a5db9-3e67-7000-8000-000000000001" format:"uuid"`
	StartsAt     time.Time                   `json:"startsAt" validate:"required" example:"2026-12-01T00:00:00Z" format:"date-time"`
	NextPeriodAt time.Time                   `json:"nextPeriodAt" validate:"required" example:"2027-01-01T00:00:00Z" format:"date-time"`
	Version      int64                       `json:"version" validate:"required" example:"1" format:"int64"`
	Current      *RecurringAllowanceRevision `json:"current" validate:"required"`
	Pending      *RecurringAllowanceRevision `json:"pending" validate:"required"`
}

// @description Possible codes: invalid_allowance, invalid_balance, invalid_request, invalid_cursor.
type AllowanceInvalid struct {
	Error AllowanceInvalidError `json:"error" validate:"required"`
}
type AllowanceInvalidError struct {
	Code    string `json:"code" validate:"required" example:"invalid_allowance"`
	Message string `json:"message" validate:"required" example:"Use a positive quantity, a nonnegative rollover cap, and a start date at midnight UTC."`
}

// @description Possible codes: allowance_not_found, balance_subject_not_found, project_environment_not_found.
type AllowanceMissing struct {
	Error AllowanceMissingError `json:"error" validate:"required"`
}
type AllowanceMissingError struct {
	Code    string `json:"code" validate:"required" example:"allowance_not_found"`
	Message string `json:"message" validate:"required" example:"No recurring allowance is configured for this customer and meter."`
}

// @description Possible codes: allowance_version_conflict, environment_inactive, balance_limit_exceeded.
type AllowanceConflict struct {
	Error AllowanceConflictError `json:"error" validate:"required"`
}
type AllowanceConflictError struct {
	Code    string `json:"code" validate:"required" example:"allowance_version_conflict"`
	Message string `json:"message" validate:"required" example:"Reload the allowance before editing. Its start date cannot change after creation."`
}

// @Summary Read a customer recurring allowance and its pending revision.
// @Description Monthly UTC renewal; month-end dates clamp without drifting. Creation grants the full current period without proration, with no credit for time before creation. Zero rollover cap resets only policy-issued quantity; a positive cap limits carry in addition to the new period quantity. Edits, including disabling, take effect next renewal; startsAt is immutable. A version of zero creates; existing policies require the latest version.
// @Tags Allowances
// @Param customerId path string true "The integrating application customer identifier." maxLength(255)
// @Param meterKey path string true "The active meter key." maxLength(120)
// @Success 200 {object} RecurringAllowance "Current and scheduled allowance configuration."
// @Failure 400 {object} AllowanceInvalid "Invalid allowance or subject keys."
// @Failure 401 {object} InvalidAPIKey "Authentication is required."
// @Failure 404 {object} AllowanceMissing "The allowance or accessible active resource was not found."
// @Failure 409 {object} AllowanceConflict "A stale revision, changed anchor, inactive environment, or quantity overflow prevents this operation."
// @Failure 500 {object} BalanceFailed "The allowance could not be loaded or saved."
// @Security projectAPIKey
// @Router /v1/customers/{customerId}/balances/{meterKey}/allowance [get]
func GetPublicRecurringAllowance() {}

// @Summary Create a monthly customer allowance or schedule its next-period revision.
// @Description Monthly UTC renewal; month-end dates clamp without drifting. Creation grants the full current period without proration, with no credit for time before creation. Zero rollover cap resets only policy-issued quantity; a positive cap limits carry in addition to the new period quantity. Edits, including disabling, take effect next renewal; startsAt is immutable. A version of zero creates; existing policies require the latest version.
// @Tags Allowances
// @Param customerId path string true "The integrating application customer identifier." maxLength(255)
// @Param meterKey path string true "The active meter key." maxLength(120)
// @Param request body models.AllowanceRequest true "Monthly allowance configuration."
// @Success 200 {object} RecurringAllowance "Current and scheduled allowance configuration."
// @Failure 400 {object} AllowanceInvalid "Invalid allowance or subject keys."
// @Failure 401 {object} InvalidAPIKey "Authentication is required."
// @Failure 404 {object} AllowanceMissing "The allowance or accessible active resource was not found."
// @Failure 409 {object} AllowanceConflict "A stale revision, changed anchor, inactive environment, or quantity overflow prevents this operation."
// @Failure 500 {object} BalanceFailed "The allowance could not be loaded or saved."
// @Security projectAPIKey
// @Router /v1/customers/{customerId}/balances/{meterKey}/allowance [put]
func PutPublicRecurringAllowance() {}

// @Summary Read a customer recurring allowance and its pending revision.
// @Description Monthly UTC renewal; month-end dates clamp without drifting. Creation grants the full current period without proration, with no credit for time before creation. Zero rollover cap resets only policy-issued quantity; a positive cap limits carry in addition to the new period quantity. Edits, including disabling, take effect next renewal; startsAt is immutable. A version of zero creates; existing policies require the latest version.
// @Tags Allowances
// @Param projectId path string true "The immutable project ID."
// @Param environment path string true "Sandbox or Live." Enums(sandbox,live)
// @Param customerId path string true "The integrating application customer identifier." maxLength(255)
// @Param meterKey path string true "The active meter key." maxLength(120)
// @Success 200 {object} RecurringAllowance "Current and scheduled allowance configuration."
// @Failure 400 {object} AllowanceInvalid "Invalid allowance or subject keys."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 404 {object} AllowanceMissing "The allowance or accessible active resource was not found."
// @Failure 409 {object} AllowanceConflict "A stale revision, changed anchor, inactive environment, or quantity overflow prevents this operation."
// @Failure 500 {object} BalanceFailed "The allowance could not be loaded or saved."
// @Security localCookieSession
// @Security productionCookieSession
// @Router /v1/projects/{projectId}/environments/{environment}/customers/{customerId}/balances/{meterKey}/allowance [get]
func GetDashboardRecurringAllowance() {}

// @Summary Create a monthly customer allowance or schedule its next-period revision.
// @Description Monthly UTC renewal; month-end dates clamp without drifting. Creation grants the full current period without proration, with no credit for time before creation. Zero rollover cap resets only policy-issued quantity; a positive cap limits carry in addition to the new period quantity. Edits, including disabling, take effect next renewal; startsAt is immutable. A version of zero creates; existing policies require the latest version.
// @Tags Allowances
// @Param projectId path string true "The immutable project ID."
// @Param environment path string true "Sandbox or Live." Enums(sandbox,live)
// @Param customerId path string true "The integrating application customer identifier." maxLength(255)
// @Param meterKey path string true "The active meter key." maxLength(120)
// @Param request body models.AllowanceRequest true "Monthly allowance configuration."
// @Success 200 {object} RecurringAllowance "Current and scheduled allowance configuration."
// @Failure 400 {object} AllowanceInvalid "Invalid allowance or subject keys."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 404 {object} AllowanceMissing "The allowance or accessible active resource was not found."
// @Failure 409 {object} AllowanceConflict "A stale revision, changed anchor, inactive environment, or quantity overflow prevents this operation."
// @Failure 500 {object} BalanceFailed "The allowance could not be loaded or saved."
// @Security localCookieSession
// @Security productionCookieSession
// @Router /v1/projects/{projectId}/environments/{environment}/customers/{customerId}/balances/{meterKey}/allowance [put]
func PutDashboardRecurringAllowance() {}

type AllowancePeriod struct {
	PeriodStart     time.Time `json:"periodStart" validate:"required" example:"2026-12-01T00:00:00Z"`
	PeriodEnd       time.Time `json:"periodEnd" validate:"required" example:"2027-01-01T00:00:00Z"`
	Revision        int64     `json:"revision" validate:"required" example:"1"`
	Quantity        int64     `json:"quantity" validate:"required" example:"1250"`
	CarriedQuantity int64     `json:"carriedQuantity" validate:"required" example:"250"`
	SkippedPeriods  int32     `json:"skippedPeriods" validate:"required" example:"0"`
	SourceType      string    `json:"sourceType" validate:"required" example:"dashboard_user" enums:"dashboard_user,api_key"`
	SourceID        uuid.UUID `json:"sourceId" validate:"required" example:"019a5db9-3e67-7000-8000-000000000002"`
}
type AllowancePeriods struct {
	Periods    []AllowancePeriod `json:"periods" validate:"required"`
	NextCursor *string           `json:"nextCursor" validate:"required"`
}

// @Summary List issued allowance periods and their rollover and recovery quantities.
// @Tags Allowances
// @Param customerId path string true "The integrating application customer identifier."
// @Param meterKey path string true "The active meter key."
// @Param cursor query string false "The next cursor from the previous response."
// @Param limit query int false "Number of periods to return." minimum(1) maximum(100) default(25)
// @Success 200 {object} AllowancePeriods "Issued periods in newest-first order."
// @Failure 400 {object} AllowanceInvalid "Invalid subject, cursor, or limit."
// @Failure 401 {object} InvalidAPIKey "Authentication is required."
// @Failure 404 {object} AllowanceMissing "The allowance or resource is unavailable."
// @Failure 409 {object} AllowanceConflict "The environment is inactive or due allowance overflows quantity capacity."
// @Failure 500 {object} BalanceFailed "The period history could not be loaded."
// @Security projectAPIKey
// @Router /v1/customers/{customerId}/balances/{meterKey}/allowance/periods [get]
func GetPublicAllowancePeriods() {}

// @Summary List issued allowance periods and their rollover and recovery quantities.
// @Tags Allowances
// @Param projectId path string true "The immutable project ID."
// @Param environment path string true "Sandbox or Live." Enums(sandbox,live)
// @Param customerId path string true "The integrating application customer identifier."
// @Param meterKey path string true "The active meter key."
// @Param cursor query string false "The next cursor from the previous response."
// @Param limit query int false "Number of periods to return." minimum(1) maximum(100) default(25)
// @Success 200 {object} AllowancePeriods "Issued periods in newest-first order."
// @Failure 400 {object} AllowanceInvalid "Invalid subject, cursor, or limit."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 404 {object} AllowanceMissing "The allowance or resource is unavailable."
// @Failure 409 {object} AllowanceConflict "The environment is inactive or due allowance overflows quantity capacity."
// @Failure 500 {object} BalanceFailed "The period history could not be loaded."
// @Security localCookieSession
// @Security productionCookieSession
// @Router /v1/projects/{projectId}/environments/{environment}/customers/{customerId}/balances/{meterKey}/allowance/periods [get]
func GetDashboardAllowancePeriods() {}
