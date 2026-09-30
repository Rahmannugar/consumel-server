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
	ErrBalanceCustomerInvalid = errors.New("balance customer ID is invalid")
	ErrBalanceMeterInvalid    = errors.New("balance meter key is invalid")
	ErrBalanceQuantityInvalid = errors.New("balance quantity is invalid")
	ErrBalanceNotFound        = errors.New("balance not found")
	ErrBalanceSubjectNotFound = errors.New("balance customer or meter not found")
	ErrBalanceOverflow        = errors.New("balance quantity exceeds the supported range")
	ErrIdempotencyKeyInvalid  = errors.New("idempotency key must be a UUID v7")
	ErrIdempotencyKeyConflict = errors.New("idempotency key was used for another request")
)

type AddBalanceRequest struct {
	CustomerID string `json:"customerId"`
	MeterKey   string `json:"meterKey"`
	Quantity   int64  `json:"quantity"`
}

type SetBalanceRequest struct {
	Quantity int64 `json:"quantity"`
}

type Balance struct {
	ID                   uuid.UUID
	ProjectEnvironmentID uuid.UUID
	CustomerID           string
	MeterKey             string
	Quantity             int64
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type Subject struct {
	CustomerID uuid.UUID
	MeterID    uuid.UUID
}

func (request AddBalanceRequest) Validate() (AddBalanceRequest, error) {
	request.CustomerID = strings.TrimSpace(request.CustomerID)
	request.MeterKey = strings.TrimSpace(request.MeterKey)
	switch {
	case request.CustomerID == "" || utf8.RuneCountInString(request.CustomerID) > customermodels.MaximumCustomerIDLength:
		return AddBalanceRequest{}, ErrBalanceCustomerInvalid
	case !metermodels.ValidMeterKey(request.MeterKey):
		return AddBalanceRequest{}, ErrBalanceMeterInvalid
	case request.Quantity <= 0:
		return AddBalanceRequest{}, ErrBalanceQuantityInvalid
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

func AddBalanceRequestOpenAPISchema() map[string]any {
	return objectSchema([]string{"customerId", "meterKey", "quantity"}, map[string]any{
		"customerId": map[string]any{"type": "string", "minLength": 1, "maxLength": customermodels.MaximumCustomerIDLength, "example": "user_123"},
		"meterKey":   map[string]any{"type": "string", "minLength": 1, "maxLength": metermodels.MaximumMeterKeyLength, "example": "api_calls"},
		"quantity":   map[string]any{"type": "integer", "format": "int64", "minimum": 1, "example": 10000},
	})
}

func SetBalanceRequestOpenAPISchema() map[string]any {
	return objectSchema([]string{"quantity"}, map[string]any{
		"quantity": map[string]any{"type": "integer", "format": "int64", "minimum": 0, "example": 10000},
	})
}

func objectSchema(required []string, properties map[string]any) map[string]any {
	return map[string]any{"type": "object", "additionalProperties": false, "required": required, "properties": properties}
}
