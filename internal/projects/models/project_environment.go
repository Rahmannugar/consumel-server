package models

import (
	"time"

	"github.com/google/uuid"
)

type ProjectEnvironmentName string

const (
	ProjectEnvironmentSandbox ProjectEnvironmentName = "sandbox"
	ProjectEnvironmentLive    ProjectEnvironmentName = "live"
)

type ProjectEnvironment struct {
	ID          uuid.UUID
	ProjectID   uuid.UUID
	Name        ProjectEnvironmentName
	ActivatedAt *time.Time
	CreatedAt   time.Time
}
