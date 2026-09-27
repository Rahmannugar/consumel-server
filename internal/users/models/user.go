package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID                uuid.UUID
	AuthlierSubjectID string
	Email             string
	CreatedAt         time.Time
}
