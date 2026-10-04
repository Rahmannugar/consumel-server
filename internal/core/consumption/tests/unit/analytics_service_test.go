package unit_test

import (
	"context"
	"errors"
	"testing"
	"time"

	consumptionmodels "github.com/Rahmannugar/consumel-server/internal/core/consumption/models"
	consumptionservices "github.com/Rahmannugar/consumel-server/internal/core/consumption/services"
	"github.com/google/uuid"
)

type analyticsRepositoryStub struct {
	buckets []consumptionmodels.AnalyticsBucket
	filter  consumptionmodels.AnalyticsFilter
}

func (repository *analyticsRepositoryStub) Aggregate(
	_ context.Context,
	_ uuid.UUID,
	filter consumptionmodels.AnalyticsFilter,
) ([]consumptionmodels.AnalyticsBucket, error) {
	repository.filter = filter
	return repository.buckets, nil
}

func TestAnalyticsServiceFillsMissingBucketsAndSummarizes(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Hour)
	from, to := now.Add(-3*time.Hour), now
	repository := &analyticsRepositoryStub{buckets: []consumptionmodels.AnalyticsBucket{
		{
			Start: from.Add(time.Hour), AcceptedOperations: 2, DeniedOperations: 1,
			AcceptedQuantity: 20, DeniedQuantity: 5, BillableOperations: 2,
		},
	}}
	service := consumptionservices.NewAnalyticsService(repository)

	result, err := service.Get(context.Background(), uuid.New(), consumptionmodels.AnalyticsFilter{
		Interval: consumptionmodels.AnalyticsIntervalHour, CustomerID: " customer_123 ",
	}, from.Format(time.RFC3339), to.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("get analytics: %v", err)
	}
	if len(result.Buckets) != 3 {
		t.Fatalf("bucket count = %d, want 3", len(result.Buckets))
	}
	if result.Buckets[0].AcceptedOperations != 0 || result.Buckets[1].AcceptedOperations != 2 {
		t.Fatalf("unexpected filled buckets: %#v", result.Buckets)
	}
	if result.Summary.AcceptedQuantity != 20 || result.Summary.DeniedQuantity != 5 {
		t.Fatalf("unexpected summary: %#v", result.Summary)
	}
	if repository.filter.CustomerID != "customer_123" {
		t.Fatalf("customer filter = %q", repository.filter.CustomerID)
	}
}

func TestAnalyticsServiceRejectsHourlyRangeLongerThan31Days(t *testing.T) {
	to := time.Now().UTC().Add(-time.Hour)
	from := to.Add(-32 * 24 * time.Hour)
	service := consumptionservices.NewAnalyticsService(&analyticsRepositoryStub{})

	_, err := service.Get(context.Background(), uuid.New(), consumptionmodels.AnalyticsFilter{
		Interval: consumptionmodels.AnalyticsIntervalHour,
	}, from.Format(time.RFC3339), to.Format(time.RFC3339))
	if !errors.Is(err, consumptionmodels.ErrAnalyticsFilterInvalid) {
		t.Fatalf("error = %v, want invalid analytics filter", err)
	}
}

func TestAnalyticsServiceRejectsInvalidResourceFilters(t *testing.T) {
	now := time.Now().UTC()
	service := consumptionservices.NewAnalyticsService(&analyticsRepositoryStub{})

	_, err := service.Get(context.Background(), uuid.New(), consumptionmodels.AnalyticsFilter{
		Interval: consumptionmodels.AnalyticsIntervalDay, MeterKey: "Not a meter key",
	}, now.Add(-24*time.Hour).Format(time.RFC3339), now.Format(time.RFC3339))
	if !errors.Is(err, consumptionmodels.ErrAnalyticsFilterInvalid) {
		t.Fatalf("error = %v, want invalid analytics filter", err)
	}
}
