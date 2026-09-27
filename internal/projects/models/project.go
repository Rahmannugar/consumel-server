package models

import (
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID               uuid.UUID
	OrganizationID   uuid.UUID
	OrganizationName string
	Name             string
	Slug             string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Environments     []ProjectEnvironment
}
