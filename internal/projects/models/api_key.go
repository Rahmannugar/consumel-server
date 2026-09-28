package models

import (
	"time"

	"github.com/google/uuid"
)

type APIKey struct {
	ID                   uuid.UUID
	ProjectEnvironmentID uuid.UUID
	Environment          ProjectEnvironmentName
	Prefix               string
	LastFour             string
	CreatedAt            time.Time
	LastUsedAt           *time.Time
	RevokedAt            *time.Time
}

type CreatedAPIKey struct {
	APIKey
	Secret string
}
