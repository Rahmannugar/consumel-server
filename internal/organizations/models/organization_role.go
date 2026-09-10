package models

import (
	"time"

	"github.com/google/uuid"
)

type OrganizationRoleSystemKey string

const (
	OrganizationRoleSystemKeyAdmin     OrganizationRoleSystemKey = "admin"
	OrganizationRoleSystemKeyDeveloper OrganizationRoleSystemKey = "developer"
)

type OrganizationRole struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	SystemKey      *OrganizationRoleSystemKey
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
}
