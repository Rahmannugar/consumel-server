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
	AggregatePortfolio(context.Context, uuid.UUID, models.PortfolioFilter) ([]models.PortfolioActivity, error)
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
	activity, err := service.repository.AggregatePortfolio(ctx, userID, filter)
	if err != nil {
		return models.Portfolio{}, err
	}
	buckets := fillPortfolioBuckets(from, to, filter.Interval, activity)
	applyPortfolioProjects(projects, activity)
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

func fillPortfolioBuckets(from, to time.Time, interval models.PortfolioInterval, activity []models.PortfolioActivity) []models.PortfolioBucket {
	buckets := make([]models.PortfolioBucket, 0)
	byStart := make(map[time.Time]int)
	for cursor := truncatePortfolioTime(from, interval); cursor.Before(to); cursor = nextPortfolioBucket(cursor, interval) {
		byStart[cursor] = len(buckets)
		buckets = append(buckets, models.PortfolioBucket{Start: cursor})
	}
	for _, row := range activity {
		index, ok := byStart[truncatePortfolioTime(row.Start, interval)]
		if !ok {
			continue
		}
		buckets[index].AllowedOperations += row.AllowedOperations
		buckets[index].BlockedOperations += row.BlockedOperations
	}
	return buckets
}

// applyPortfolioProjects sums each project's buckets into its comparison totals.
// Projects without activity keep their zero values and nil last activity.
func applyPortfolioProjects(projects []models.PortfolioProject, activity []models.PortfolioActivity) {
	if len(projects) == 0 {
		return
	}
	byProject := make(map[uuid.UUID]int, len(projects))
	for index := range projects {
		byProject[projects[index].ID] = index
	}
	for _, row := range activity {
		index, ok := byProject[row.ProjectID]
		if !ok {
			continue
		}
		project := &projects[index]
		project.AllowedOperations += row.AllowedOperations
		project.BlockedOperations += row.BlockedOperations
		if project.LastActivityAt == nil || row.LastActivityAt.After(*project.LastActivityAt) {
			lastActivity := row.LastActivityAt
			project.LastActivityAt = &lastActivity
		}
	}
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
