package models

import (
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Environments   []ProjectEnvironment
}
