package models

import (
	"time"

	"github.com/google/uuid"
)

type Organization struct {
	ID          uuid.UUID
	OwnerUserID uuid.UUID
	Name        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
	SuspendedAt *time.Time
}
