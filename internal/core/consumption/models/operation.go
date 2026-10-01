package models

import (
	"errors"
	"time"

	metermodels "github.com/Rahmannugar/consumel-server/internal/meters/models"
	"github.com/google/uuid"
)

var (
	ErrOperationCursorInvalid = errors.New("operation cursor is invalid")
	ErrOperationFilterInvalid = errors.New("operation filter is invalid")
)

type OperationStatus string

const (
	OperationStatusAccepted OperationStatus = "accepted"
	OperationStatusDenied   OperationStatus = "denied"
)

type Operation struct {
	ID               uuid.UUID
	CustomerID       string
	MeterKey         string
	Quantity         int64
	MeterType        metermodels.MeterType
	Status           OperationStatus
	DenialReason     *string
	BalanceDebited   int64
	RemainingBalance *int64
	Billable         bool
	ReplayCount      int64
	LastReplayedAt   *time.Time
	CreatedAt        time.Time
}

type OperationListCursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

type OperationListFilter struct {
	Status     OperationStatus
	CustomerID string
	MeterKey   string
	From       time.Time
	To         time.Time
}
