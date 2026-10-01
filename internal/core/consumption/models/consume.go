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
	CustomerID string `json:"customerId" validate:"required,min=1,max=255" example:"customer_123"`
	MeterKey   string `json:"meterKey" validate:"required,min=1,max=120" pattern:"^[a-z][a-z0-9_-]*$" example:"api_calls"`
	Quantity   int64  `json:"quantity" validate:"required,min=1" format:"int64" example:"500"`
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
