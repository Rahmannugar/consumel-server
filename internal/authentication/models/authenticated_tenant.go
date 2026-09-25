package models

import (
	"time"

	organizationmodels "github.com/Rahmannugar/consumel-server/internal/organizations/models"
	usermodels "github.com/Rahmannugar/consumel-server/internal/users/models"
)

type Session struct {
	ID        string
	SubjectID string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type AuthenticatedTenant struct {
	Session            Session
	User               usermodels.User
	OrganizationAccess []organizationmodels.OrganizationAccess
}
