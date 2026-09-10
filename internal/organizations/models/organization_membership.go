package models

import (
	"time"

	"github.com/google/uuid"
)

type OrganizationMembershipRole string

const (
	OrganizationMembershipRoleAdmin     OrganizationMembershipRole = "admin"
	OrganizationMembershipRoleDeveloper OrganizationMembershipRole = "developer"
)

type OrganizationMembershipStatus string

const (
	OrganizationMembershipStatusActive    OrganizationMembershipStatus = "active"
	OrganizationMembershipStatusSuspended OrganizationMembershipStatus = "suspended"
)

type OrganizationMembership struct {
	OrganizationID uuid.UUID
	UserID         uuid.UUID
	Role           OrganizationMembershipRole
	Status         OrganizationMembershipStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
	RemovedAt      *time.Time
}
