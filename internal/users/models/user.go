package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID
	ClerkUserID string
	CreatedAt   time.Time
}
