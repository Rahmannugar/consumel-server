package models

import "github.com/google/uuid"

type OrganizationAccess struct {
	OrganizationID   uuid.UUID
	OrganizationName string
	Owner            bool
	RoleID           uuid.UUID
	RoleName         string
	RoleSystemKey    *OrganizationRoleSystemKey
}
