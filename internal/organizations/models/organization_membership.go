package models

import (
	"time"

	"github.com/google/uuid"
)

type OrganizationMembershipStatus string

const (
	OrganizationMembershipStatusActive    OrganizationMembershipStatus = "active"
	OrganizationMembershipStatusSuspended OrganizationMembershipStatus = "suspended"
)

type OrganizationMembership struct {
	OrganizationID uuid.UUID
	UserID         uuid.UUID
	RoleID         uuid.UUID
	Status         OrganizationMembershipStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
	RemovedAt      *time.Time
}
