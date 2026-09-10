package models

import (
	"time"

	"github.com/google/uuid"
)

type OrganizationMembershipRole string

const (
	OrganizationMembershipRoleOwner           OrganizationMembershipRole = "owner"
	OrganizationMembershipRoleDeveloper       OrganizationMembershipRole = "developer"
	OrganizationMembershipRoleFinance         OrganizationMembershipRole = "finance"
	OrganizationMembershipRoleCustomerSupport OrganizationMembershipRole = "customer_support"
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
}
