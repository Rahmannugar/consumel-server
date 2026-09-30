package repositories

import (
	"context"
	"fmt"
	"time"

	consumptionmodels "github.com/Rahmannugar/consumel-server/internal/core/consumption/models"
	consumptiondb "github.com/Rahmannugar/consumel-server/internal/core/consumption/repositories/generated"
	metermodels "github.com/Rahmannugar/consumel-server/internal/meters/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OperationRepository struct {
	queries *consumptiondb.Queries
}

func NewOperationRepository(pool *pgxpool.Pool) *OperationRepository {
	return &OperationRepository{queries: consumptiondb.New(pool)}
}

func (repository *OperationRepository) List(
	ctx context.Context,
	projectEnvironmentID uuid.UUID,
	cursor *consumptionmodels.OperationListCursor,
	limit int,
	filter consumptionmodels.OperationListFilter,
) ([]consumptionmodels.Operation, *consumptionmodels.OperationListCursor, error) {
	params := consumptiondb.ListConsumptionOperationsParams{
		ProjectEnvironmentID: projectEnvironmentID,
		FromTime:             pgtype.Timestamptz{Time: filter.From, Valid: true},
		ToTime:               pgtype.Timestamptz{Time: filter.To, Valid: true},
		StatusFilter:         string(filter.Status),
		PageSize:             int32(limit + 1),
	}
	if cursor != nil {
		params.CursorCreatedAt = pgtype.Timestamptz{Time: cursor.CreatedAt, Valid: true}
		params.CursorID = pgtype.UUID{Bytes: cursor.ID, Valid: true}
	}
	rows, err := repository.queries.ListConsumptionOperations(ctx, params)
	if err != nil {
		return nil, nil, fmt.Errorf("list consumption operations: %w", err)
	}
	operations := make([]consumptionmodels.Operation, 0, min(len(rows), limit))
	for index, row := range rows {
		if index == limit {
			last := operations[len(operations)-1]
			return operations, &consumptionmodels.OperationListCursor{CreatedAt: last.CreatedAt, ID: last.ID}, nil
		}
		operations = append(operations, operationModel(row))
	}
	return operations, nil, nil
}

func (repository *OperationRepository) Get(
	ctx context.Context,
	projectEnvironmentID, operationID uuid.UUID,
) (consumptionmodels.Operation, error) {
	row, err := repository.queries.ConsumptionOperationByID(ctx, consumptiondb.ConsumptionOperationByIDParams{
		ProjectEnvironmentID: projectEnvironmentID,
		ID:                   operationID,
	})
	if err != nil {
		return consumptionmodels.Operation{}, err
	}
	return operationModel(row), nil
}

func operationModel(row consumptiondb.ConsumptionOperation) consumptionmodels.Operation {
	return consumptionmodels.Operation{
		ID: row.ID, CustomerID: row.RequestCustomerID, MeterKey: row.RequestMeterKey,
		Quantity: row.RequestedQuantity, MeterType: metermodels.MeterType(row.MeterType),
		Status: consumptionmodels.OperationStatus(row.Status), DenialReason: row.DenialReason,
		BalanceDebited: row.BalanceDebited, RemainingBalance: row.ResultingBalance,
		Billable: row.Billable, ReplayCount: row.ReplayCount,
		LastReplayedAt: timePointer(row.LastReplayedAt), CreatedAt: row.CreatedAt.Time,
	}
}

func timePointer(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}
