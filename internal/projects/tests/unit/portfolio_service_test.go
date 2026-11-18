package unit_test

import (
	"context"
	"errors"
	"testing"
	"time"

	projectmodels "github.com/Rahmannugar/consumel-server/internal/projects/models"
	projectservices "github.com/Rahmannugar/consumel-server/internal/projects/services"
	"github.com/google/uuid"
)

type portfolioRepositoryStub struct {
	projects []projectmodels.PortfolioProject
	activity []projectmodels.PortfolioActivity
}

func (repository *portfolioRepositoryStub) ProjectPortfolio(
	context.Context,
	uuid.UUID,
	projectmodels.PortfolioFilter,
) ([]projectmodels.PortfolioProject, error) {
	return repository.projects, nil
}

func (repository *portfolioRepositoryStub) AggregatePortfolio(
	context.Context,
	uuid.UUID,
	projectmodels.PortfolioFilter,
) ([]projectmodels.PortfolioActivity, error) {
	return repository.activity, nil
}

func TestPortfolioPreservesEmptyCalendarBucketsAndTotalsOutcomes(t *testing.T) {
	from := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)
	firstID := uuid.New()
	secondID := uuid.New()
	service := projectservices.NewPortfolioService(&portfolioRepositoryStub{
		projects: []projectmodels.PortfolioProject{
			{ID: firstID, Name: "First API", Slug: "first-api"},
			{ID: secondID, Name: "Second API", Slug: "second-api"},
		},
		activity: []projectmodels.PortfolioActivity{
			{ProjectID: firstID, Start: from, AllowedOperations: 7, BlockedOperations: 1, LastActivityAt: from.Add(time.Hour)},
			{ProjectID: firstID, Start: from.AddDate(0, 0, 2), AllowedOperations: 3, BlockedOperations: 2, LastActivityAt: from.AddDate(0, 0, 2).Add(2 * time.Hour)},
		},
	})

	portfolio, err := service.Get(
		t.Context(), uuid.New(),
		projectmodels.PortfolioFilter{
			Environment: projectmodels.ProjectEnvironmentSandbox,
			Interval:    projectmodels.PortfolioIntervalDay,
		},
		from.Format(time.RFC3339),
		from.AddDate(0, 0, 3).Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("get portfolio: %v", err)
	}
	if len(portfolio.Buckets) != 3 {
		t.Fatalf("bucket count = %d, want 3", len(portfolio.Buckets))
	}
	if portfolio.Buckets[1].AllowedOperations != 0 || portfolio.Buckets[1].BlockedOperations != 0 {
		t.Fatalf("empty bucket = %#v, want zero outcomes", portfolio.Buckets[1])
	}
	if portfolio.Summary.AllowedOperations != 10 || portfolio.Summary.BlockedOperations != 3 {
		t.Fatalf("summary = %#v, want 10 allowed and 3 blocked", portfolio.Summary)
	}
	first := portfolio.Projects[0]
	if first.AllowedOperations != 10 || first.BlockedOperations != 3 {
		t.Fatalf("first project totals = %#v, want 10 allowed and 3 blocked", first)
	}
	if first.LastActivityAt == nil || !first.LastActivityAt.Equal(from.AddDate(0, 0, 2).Add(2*time.Hour)) {
		t.Fatalf("first project last activity = %v, want the latest bucket", first.LastActivityAt)
	}
	second := portfolio.Projects[1]
	if second.AllowedOperations != 0 || second.BlockedOperations != 0 || second.LastActivityAt != nil {
		t.Fatalf("project without activity = %#v, want zero totals and no last activity", second)
	}
}

func TestPortfolioRejectsMixedOrUnboundedScope(t *testing.T) {
	service := projectservices.NewPortfolioService(&portfolioRepositoryStub{})
	now := time.Now().UTC()

	_, err := service.Get(
		t.Context(), uuid.New(),
		projectmodels.PortfolioFilter{
			Environment: "all",
			Interval:    projectmodels.PortfolioIntervalDay,
		},
		now.Add(-24*time.Hour).Format(time.RFC3339),
		now.Format(time.RFC3339),
	)
	if !errors.Is(err, projectmodels.ErrPortfolioFilterInvalid) {
		t.Fatalf("mixed environment error = %v, want invalid filter", err)
	}

	_, err = service.Get(
		t.Context(), uuid.New(),
		projectmodels.PortfolioFilter{
			Environment: projectmodels.ProjectEnvironmentLive,
			Interval:    projectmodels.PortfolioIntervalDay,
		},
		now.AddDate(-1, 0, -1).Format(time.RFC3339),
		now.Format(time.RFC3339),
	)
	if !errors.Is(err, projectmodels.ErrPortfolioFilterInvalid) {
		t.Fatalf("unbounded range error = %v, want invalid filter", err)
	}
}
