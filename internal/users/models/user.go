package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID                uuid.UUID
	AuthlierSubjectID string
	CreatedAt         time.Time
}
