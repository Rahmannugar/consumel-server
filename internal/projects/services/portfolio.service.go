package services

import (
	"context"
	"time"

	"github.com/Rahmannugar/consumel-server/internal/projects/models"
	"github.com/google/uuid"
)

const (
	maximumPortfolioRange       = 365 * 24 * time.Hour
	maximumHourlyPortfolioRange = 31 * 24 * time.Hour
)

type PortfolioRepository interface {
	ProjectPortfolio(context.Context, uuid.UUID, models.PortfolioFilter) ([]models.PortfolioProject, error)
	AggregatePortfolio(context.Context, uuid.UUID, models.PortfolioFilter) ([]models.PortfolioBucket, error)
}

type PortfolioService struct {
	repository PortfolioRepository
	clock      func() time.Time
}

func NewPortfolioService(repository PortfolioRepository) *PortfolioService {
	return &PortfolioService{repository: repository, clock: time.Now}
}

func (service *PortfolioService) Get(
	ctx context.Context,
	userID uuid.UUID,
	filter models.PortfolioFilter,
	fromValue string,
	toValue string,
) (models.Portfolio, error) {
	if userID == uuid.Nil ||
		(filter.Environment != models.ProjectEnvironmentSandbox && filter.Environment != models.ProjectEnvironmentLive) ||
		(filter.Interval != models.PortfolioIntervalHour && filter.Interval != models.PortfolioIntervalDay) {
		return models.Portfolio{}, models.ErrPortfolioFilterInvalid
	}

	from, to, err := portfolioRange(service.clock().UTC(), fromValue, toValue, filter.Interval)
	if err != nil {
		return models.Portfolio{}, models.ErrPortfolioFilterInvalid
	}
	filter.From, filter.To = from, to

	projects, err := service.repository.ProjectPortfolio(ctx, userID, filter)
	if err != nil {
		return models.Portfolio{}, err
	}
	stored, err := service.repository.AggregatePortfolio(ctx, userID, filter)
	if err != nil {
		return models.Portfolio{}, err
	}
	buckets := fillPortfolioBuckets(from, to, filter.Interval, stored)
	summary := models.PortfolioBucket{}
	for _, bucket := range buckets {
		summary.AllowedOperations += bucket.AllowedOperations
		summary.BlockedOperations += bucket.BlockedOperations
	}

	return models.Portfolio{
		Environment: filter.Environment,
		From:        from,
		To:          to,
		Interval:    filter.Interval,
		Summary:     summary,
		Buckets:     buckets,
		Projects:    projects,
	}, nil
}

func portfolioRange(now time.Time, fromValue, toValue string, interval models.PortfolioInterval) (time.Time, time.Time, error) {
	from, err := time.Parse(time.RFC3339, fromValue)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	to, err := time.Parse(time.RFC3339, toValue)
	if err != nil || !from.Before(to) || to.After(now.Add(5*time.Minute)) {
		return time.Time{}, time.Time{}, models.ErrPortfolioFilterInvalid
	}
	maximum := maximumPortfolioRange
	if interval == models.PortfolioIntervalHour {
		maximum = maximumHourlyPortfolioRange
	}
	if to.Sub(from) > maximum {
		return time.Time{}, time.Time{}, models.ErrPortfolioFilterInvalid
	}
	return from.UTC(), to.UTC(), nil
}

func fillPortfolioBuckets(from, to time.Time, interval models.PortfolioInterval, stored []models.PortfolioBucket) []models.PortfolioBucket {
	byStart := make(map[time.Time]models.PortfolioBucket, len(stored))
	for _, bucket := range stored {
		byStart[bucket.Start.UTC()] = bucket
	}
	start := truncatePortfolioTime(from, interval)
	buckets := make([]models.PortfolioBucket, 0)
	for cursor := start; cursor.Before(to); cursor = nextPortfolioBucket(cursor, interval) {
		bucket, ok := byStart[cursor]
		if !ok {
			bucket = models.PortfolioBucket{Start: cursor}
		}
		buckets = append(buckets, bucket)
	}
	return buckets
}

func truncatePortfolioTime(value time.Time, interval models.PortfolioInterval) time.Time {
	value = value.UTC()
	if interval == models.PortfolioIntervalHour {
		return value.Truncate(time.Hour)
	}
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func nextPortfolioBucket(value time.Time, interval models.PortfolioInterval) time.Time {
	if interval == models.PortfolioIntervalHour {
		return value.Add(time.Hour)
	}
	return value.AddDate(0, 0, 1)
}
