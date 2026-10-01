package models

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	customermodels "github.com/Rahmannugar/consumel-server/internal/customers/models"
	metermodels "github.com/Rahmannugar/consumel-server/internal/meters/models"
	"github.com/google/uuid"
)

var (
	ErrBalanceCustomerInvalid   = errors.New("balance customer ID is invalid")
	ErrBalanceMeterInvalid      = errors.New("balance meter key is invalid")
	ErrBalanceQuantityInvalid   = errors.New("balance quantity is invalid")
	ErrBalanceExpirationInvalid = errors.New("balance expiration must be in the future")
	ErrBalanceNotFound          = errors.New("balance not found")
	ErrBalanceSubjectNotFound   = errors.New("balance customer or meter not found")
	ErrBalanceOverflow          = errors.New("balance quantity exceeds the supported range")
	ErrIdempotencyKeyInvalid    = errors.New("idempotency key must be a UUID v7")
	ErrIdempotencyKeyConflict   = errors.New("idempotency key was used for another request")
)

type AddBalanceRequest struct {
	CustomerID string `json:"customerId" validate:"required,min=1,max=255" example:"user_123"`
	MeterKey   string `json:"meterKey" validate:"required,min=1,max=120" example:"api_calls"`
	Quantity   int64  `json:"quantity" validate:"required,min=1" format:"int64" example:"10000"`
	// Optional instant when the added entitlement stops being available.
	ExpiresAt *time.Time `json:"expiresAt" format:"date-time" example:"2026-10-31T00:00:00Z"`
}

type SetBalanceRequest struct {
	Quantity int64 `json:"quantity" validate:"required,min=0" format:"int64" example:"10000"`
}

type BalanceMutationSource struct {
	Type    string
	ActorID uuid.UUID
}

const (
	BalanceSourceAPIKey        = "api_key"
	BalanceSourceDashboardUser = "dashboard_user"
)

type Balance struct {
	ID                   uuid.UUID
	ProjectEnvironmentID uuid.UUID
	CustomerID           string
	MeterKey             string
	Quantity             int64
	NextExpiresAt        *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type EntitlementGrantStatus string

const (
	EntitlementGrantActive    EntitlementGrantStatus = "active"
	EntitlementGrantExhausted EntitlementGrantStatus = "exhausted"
	EntitlementGrantExpired   EntitlementGrantStatus = "expired"
)

type EntitlementGrant struct {
	ID                uuid.UUID
	GrantedQuantity   int64
	RemainingQuantity int64
	Status            EntitlementGrantStatus
	ExpiresAt         *time.Time
	CreatedAt         time.Time
}

type BalanceActivity struct {
	ID                uuid.UUID
	Kind              string
	QuantityChange    int64
	ResultingQuantity *int64
	ExpiresAt         *time.Time
	SourceType        string
	SourceID          *uuid.UUID
	OccurredAt        time.Time
}

type Subject struct {
	CustomerID uuid.UUID
	MeterID    uuid.UUID
}

func (request AddBalanceRequest) Validate() (AddBalanceRequest, error) {
	return request.ValidateAt(time.Now())
}

func (request AddBalanceRequest) ValidateAt(now time.Time) (AddBalanceRequest, error) {
	request.CustomerID = strings.TrimSpace(request.CustomerID)
	request.MeterKey = strings.TrimSpace(request.MeterKey)
	if request.ExpiresAt != nil {
		normalized := request.ExpiresAt.UTC().Truncate(time.Microsecond)
		request.ExpiresAt = &normalized
	}
	switch {
	case request.CustomerID == "" || utf8.RuneCountInString(request.CustomerID) > customermodels.MaximumCustomerIDLength:
		return AddBalanceRequest{}, ErrBalanceCustomerInvalid
	case !metermodels.ValidMeterKey(request.MeterKey):
		return AddBalanceRequest{}, ErrBalanceMeterInvalid
	case request.Quantity <= 0:
		return AddBalanceRequest{}, ErrBalanceQuantityInvalid
	case request.ExpiresAt != nil && !request.ExpiresAt.After(now):
		return AddBalanceRequest{}, ErrBalanceExpirationInvalid
	default:
		return request, nil
	}
}

func (request SetBalanceRequest) Validate() (SetBalanceRequest, error) {
	if request.Quantity < 0 {
		return SetBalanceRequest{}, ErrBalanceQuantityInvalid
	}
	return request, nil
}
