package repositories

import (
	"context"
	"fmt"

	consumptionmodels "github.com/Rahmannugar/consumel-server/internal/core/consumption/models"
	consumptiondb "github.com/Rahmannugar/consumel-server/internal/core/consumption/repositories/generated"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AnalyticsRepository struct {
	queries *consumptiondb.Queries
}

func NewAnalyticsRepository(pool *pgxpool.Pool) *AnalyticsRepository {
	return &AnalyticsRepository{queries: consumptiondb.New(pool)}
}

func (repository *AnalyticsRepository) Aggregate(
	ctx context.Context,
	projectEnvironmentID uuid.UUID,
	filter consumptionmodels.AnalyticsFilter,
) ([]consumptionmodels.AnalyticsBucket, error) {
	rows, err := repository.queries.AggregateConsumptionOperations(ctx, consumptiondb.AggregateConsumptionOperationsParams{
		BucketInterval:       string(filter.Interval),
		ProjectEnvironmentID: projectEnvironmentID,
		FromTime:             pgtype.Timestamptz{Time: filter.From, Valid: true},
		ToTime:               pgtype.Timestamptz{Time: filter.To, Valid: true},
		CustomerFilter:       filter.CustomerID,
		MeterFilter:          filter.MeterKey,
	})
	if err != nil {
		return nil, fmt.Errorf("aggregate consumption operations: %w", err)
	}
	buckets := make([]consumptionmodels.AnalyticsBucket, 0, len(rows))
	for _, row := range rows {
		buckets = append(buckets, consumptionmodels.AnalyticsBucket{
			Start:              row.BucketStart.Time,
			AcceptedOperations: row.AcceptedOperations,
			DeniedOperations:   row.DeniedOperations,
			AcceptedQuantity:   row.AcceptedQuantity,
			DeniedQuantity:     row.DeniedQuantity,
			BillableOperations: row.BillableOperations,
		})
	}
	return buckets, nil
}
