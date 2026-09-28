package models

import "github.com/google/uuid"

type APIKeyContext struct {
	APIKeyID             uuid.UUID
	OrganizationID       uuid.UUID
	ProjectID            uuid.UUID
	ProjectEnvironmentID uuid.UUID
	Environment          string
}
