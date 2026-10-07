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
