package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrPortfolioFilterInvalid = errors.New("project portfolio filter is invalid")

type PortfolioInterval string

const (
	PortfolioIntervalHour PortfolioInterval = "hour"
	PortfolioIntervalDay  PortfolioInterval = "day"
)

type PortfolioFilter struct {
	Environment ProjectEnvironmentName
	Interval    PortfolioInterval
	From        time.Time
	To          time.Time
}

type PortfolioBucket struct {
	Start             time.Time
	AllowedOperations int64
	BlockedOperations int64
}

// PortfolioActivity is one project's outcomes within one calendar bucket. The
// portfolio service derives both the organization buckets and the per-project
// totals from these rows so the operation table is scanned once.
type PortfolioActivity struct {
	ProjectID         uuid.UUID
	Start             time.Time
	AllowedOperations int64
	BlockedOperations int64
	LastActivityAt    time.Time
}

type PortfolioProject struct {
	ID                uuid.UUID
	Name              string
	Slug              string
	EnvironmentID     uuid.UUID
	ActivatedAt       *time.Time
	AllowedOperations int64
	BlockedOperations int64
	LastActivityAt    *time.Time
}

type Portfolio struct {
	Environment ProjectEnvironmentName
	From        time.Time
	To          time.Time
	Interval    PortfolioInterval
	Summary     PortfolioBucket
	Buckets     []PortfolioBucket
	Projects    []PortfolioProject
}
