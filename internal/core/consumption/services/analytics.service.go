package services

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	consumptionmodels "github.com/Rahmannugar/consumel-server/internal/core/consumption/models"
	customermodels "github.com/Rahmannugar/consumel-server/internal/customers/models"
	metermodels "github.com/Rahmannugar/consumel-server/internal/meters/models"
	"github.com/google/uuid"
)

const maximumHourlyAnalyticsRange = 31 * 24 * time.Hour

type AnalyticsRepository interface {
	Aggregate(context.Context, uuid.UUID, consumptionmodels.AnalyticsFilter) ([]consumptionmodels.AnalyticsBucket, error)
}

type AnalyticsService struct {
	repository AnalyticsRepository
	clock      func() time.Time
}

func NewAnalyticsService(repository AnalyticsRepository) *AnalyticsService {
	return &AnalyticsService{repository: repository, clock: time.Now}
}

func (service *AnalyticsService) Get(
	ctx context.Context,
	projectEnvironmentID uuid.UUID,
	filter consumptionmodels.AnalyticsFilter,
	fromValue, toValue string,
) (consumptionmodels.Analytics, error) {
	filter.CustomerID = strings.TrimSpace(filter.CustomerID)
	if utf8.RuneCountInString(filter.CustomerID) > customermodels.MaximumCustomerIDLength {
		return consumptionmodels.Analytics{}, consumptionmodels.ErrAnalyticsFilterInvalid
	}
	filter.MeterKey = strings.TrimSpace(filter.MeterKey)
	if filter.MeterKey != "" && !metermodels.ValidMeterKey(filter.MeterKey) {
		return consumptionmodels.Analytics{}, consumptionmodels.ErrAnalyticsFilterInvalid
	}
	if filter.Interval != consumptionmodels.AnalyticsIntervalHour && filter.Interval != consumptionmodels.AnalyticsIntervalDay {
		return consumptionmodels.Analytics{}, consumptionmodels.ErrAnalyticsFilterInvalid
	}
	from, to, err := analyticsRange(service.clock().UTC(), fromValue, toValue, filter.Interval)
	if err != nil {
		return consumptionmodels.Analytics{}, consumptionmodels.ErrAnalyticsFilterInvalid
	}
	filter.From, filter.To = from, to
	stored, err := service.repository.Aggregate(ctx, projectEnvironmentID, filter)
	if err != nil {
		return consumptionmodels.Analytics{}, err
	}
	buckets := fillAnalyticsBuckets(from, to, filter.Interval, stored)
	summary := consumptionmodels.AnalyticsSummary{}
	for _, bucket := range buckets {
		summary.AcceptedOperations += bucket.AcceptedOperations
		summary.DeniedOperations += bucket.DeniedOperations
		summary.AcceptedQuantity += bucket.AcceptedQuantity
		summary.DeniedQuantity += bucket.DeniedQuantity
		summary.BillableOperations += bucket.BillableOperations
	}
	return consumptionmodels.Analytics{
		From: from, To: to, Interval: filter.Interval, Summary: summary, Buckets: buckets,
	}, nil
}

func analyticsRange(now time.Time, fromValue, toValue string, interval consumptionmodels.AnalyticsInterval) (time.Time, time.Time, error) {
	if fromValue == "" || toValue == "" {
		return time.Time{}, time.Time{}, consumptionmodels.ErrAnalyticsFilterInvalid
	}
	from, err := time.Parse(time.RFC3339, fromValue)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	to, err := time.Parse(time.RFC3339, toValue)
	if err != nil || !from.Before(to) || to.After(now.Add(5*time.Minute)) {
		return time.Time{}, time.Time{}, consumptionmodels.ErrAnalyticsFilterInvalid
	}
	maximum := maximumOperationRange
	if interval == consumptionmodels.AnalyticsIntervalHour {
		maximum = maximumHourlyAnalyticsRange
	}
	if to.Sub(from) > maximum {
		return time.Time{}, time.Time{}, consumptionmodels.ErrAnalyticsFilterInvalid
	}
	return from.UTC(), to.UTC(), nil
}

func fillAnalyticsBuckets(from, to time.Time, interval consumptionmodels.AnalyticsInterval, stored []consumptionmodels.AnalyticsBucket) []consumptionmodels.AnalyticsBucket {
	byStart := make(map[time.Time]consumptionmodels.AnalyticsBucket, len(stored))
	for _, bucket := range stored {
		byStart[bucket.Start.UTC()] = bucket
	}
	start := truncateAnalyticsTime(from, interval)
	buckets := make([]consumptionmodels.AnalyticsBucket, 0)
	for cursor := start; cursor.Before(to); cursor = nextAnalyticsBucket(cursor, interval) {
		bucket, ok := byStart[cursor]
		if !ok {
			bucket = consumptionmodels.AnalyticsBucket{Start: cursor}
		}
		buckets = append(buckets, bucket)
	}
	return buckets
}

func truncateAnalyticsTime(value time.Time, interval consumptionmodels.AnalyticsInterval) time.Time {
	value = value.UTC()
	if interval == consumptionmodels.AnalyticsIntervalHour {
		return value.Truncate(time.Hour)
	}
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func nextAnalyticsBucket(value time.Time, interval consumptionmodels.AnalyticsInterval) time.Time {
	if interval == consumptionmodels.AnalyticsIntervalHour {
		return value.Add(time.Hour)
	}
	return value.AddDate(0, 0, 1)
}
