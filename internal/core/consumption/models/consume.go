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
	ErrConsumeCustomerInvalid = errors.New("consume customer ID is invalid")
	ErrConsumeMeterInvalid    = errors.New("consume meter key is invalid")
	ErrConsumeQuantityInvalid = errors.New("consume quantity is invalid")
	ErrConsumeMeterNotFound   = errors.New("consume meter not found")
	ErrInsufficientBalance    = errors.New("insufficient balance")
)

type ConsumeRequest struct {
	CustomerID string `json:"customerId"`
	MeterKey   string `json:"meterKey"`
	Quantity   int64  `json:"quantity"`
}

type UsageEvent struct {
	ID                   uuid.UUID
	ProjectEnvironmentID uuid.UUID
	CustomerID           string
	MeterKey             string
	Quantity             int64
	MeterType            metermodels.MeterType
	BalanceDebited       int64
	RemainingBalance     *int64
	Billable             bool
	CreatedAt            time.Time
}

func (request ConsumeRequest) Validate() (ConsumeRequest, error) {
	request.CustomerID = strings.TrimSpace(request.CustomerID)
	request.MeterKey = strings.TrimSpace(request.MeterKey)
	switch {
	case request.CustomerID == "" || utf8.RuneCountInString(request.CustomerID) > customermodels.MaximumCustomerIDLength:
		return ConsumeRequest{}, ErrConsumeCustomerInvalid
	case !metermodels.ValidMeterKey(request.MeterKey):
		return ConsumeRequest{}, ErrConsumeMeterInvalid
	case request.Quantity <= 0:
		return ConsumeRequest{}, ErrConsumeQuantityInvalid
	default:
		return request, nil
	}
}

func ConsumeRequestOpenAPISchema() map[string]any {
	return objectSchema([]string{"customerId", "meterKey", "quantity"}, map[string]any{
		"customerId": map[string]any{"type": "string", "minLength": 1, "maxLength": customermodels.MaximumCustomerIDLength, "example": "customer_123"},
		"meterKey":   map[string]any{"type": "string", "minLength": 1, "maxLength": metermodels.MaximumMeterKeyLength, "pattern": `^[a-z][a-z0-9_-]*$`, "example": "api_calls"},
		"quantity":   map[string]any{"type": "integer", "format": "int64", "minimum": 1, "example": 500},
	})
}
