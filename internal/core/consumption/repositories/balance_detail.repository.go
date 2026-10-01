package repositories

import (
	"context"
	"fmt"
	"time"

	consumptionmodels "github.com/Rahmannugar/consumel-server/internal/core/consumption/models"
	consumptiondb "github.com/Rahmannugar/consumel-server/internal/core/consumption/repositories/generated"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func (repository *BalanceRepository) ListGrants(
	ctx context.Context,
	projectEnvironmentID uuid.UUID,
	customerID, meterKey string,
	cursor *consumptionmodels.OperationListCursor,
	limit int,
) ([]consumptionmodels.EntitlementGrant, *consumptionmodels.OperationListCursor, error) {
	queries := consumptiondb.New(repository.pool)
	params := consumptiondb.ListEntitlementGrantsParams{
		ProjectEnvironmentID: projectEnvironmentID,
		CustomerID:           customerID,
		MeterKey:             meterKey,
		PageSize:             int32(limit + 1),
	}
	if cursor != nil {
		params.CursorCreatedAt = pgtype.Timestamptz{Time: cursor.CreatedAt, Valid: true}
		params.CursorID = pgtype.UUID{Bytes: cursor.ID, Valid: true}
	}
	rows, err := queries.ListEntitlementGrants(ctx, params)
	if err != nil {
		return nil, nil, fmt.Errorf("list entitlement grants: %w", err)
	}
	grants := make([]consumptionmodels.EntitlementGrant, 0, min(len(rows), limit))
	for index, row := range rows {
		if index == limit {
			last := grants[len(grants)-1]
			return grants, &consumptionmodels.OperationListCursor{CreatedAt: last.CreatedAt, ID: last.ID}, nil
		}
		grants = append(grants, entitlementGrantModel(row))
	}
	if len(grants) == 0 {
		if _, err := repository.Get(ctx, projectEnvironmentID, customerID, meterKey); err != nil {
			return nil, nil, err
		}
	}
	return grants, nil, nil
}

func (repository *BalanceRepository) ListActivity(
	ctx context.Context,
	projectEnvironmentID uuid.UUID,
	customerID, meterKey string,
	cursor *consumptionmodels.OperationListCursor,
	limit int,
) ([]consumptionmodels.BalanceActivity, *consumptionmodels.OperationListCursor, error) {
	queries := consumptiondb.New(repository.pool)
	params := consumptiondb.ListBalanceActivityParams{
		ProjectEnvironmentID: projectEnvironmentID,
		CustomerID:           customerID,
		MeterKey:             meterKey,
		PageSize:             int32(limit + 1),
	}
	if cursor != nil {
		params.CursorCreatedAt = pgtype.Timestamptz{Time: cursor.CreatedAt, Valid: true}
		params.CursorID = pgtype.UUID{Bytes: cursor.ID, Valid: true}
	}
	rows, err := queries.ListBalanceActivity(ctx, params)
	if err != nil {
		return nil, nil, fmt.Errorf("list balance activity: %w", err)
	}
	activity := make([]consumptionmodels.BalanceActivity, 0, min(len(rows), limit))
	for index, row := range rows {
		if index == limit {
			last := activity[len(activity)-1]
			return activity, &consumptionmodels.OperationListCursor{CreatedAt: last.OccurredAt, ID: last.ID}, nil
		}
		activity = append(activity, consumptionmodels.BalanceActivity{
			ID: row.ID, Kind: row.Kind, QuantityChange: row.QuantityChange,
			ResultingQuantity: row.ResultingQuantity, ExpiresAt: entitlementTimePointer(row.ExpiresAt),
			SourceType: row.SourceType, SourceID: uuidPointer(row.SourceID), OccurredAt: row.OccurredAt.Time,
		})
	}
	if len(activity) == 0 {
		if _, err := repository.Get(ctx, projectEnvironmentID, customerID, meterKey); err != nil {
			return nil, nil, err
		}
	}
	return activity, nil, nil
}

func uuidPointer(value pgtype.UUID) *uuid.UUID {
	if !value.Valid {
		return nil
	}
	id := uuid.UUID(value.Bytes)
	return &id
}

func entitlementGrantModel(row consumptiondb.EntitlementGrant) consumptionmodels.EntitlementGrant {
	status := consumptionmodels.EntitlementGrantActive
	if row.RemainingQuantity == 0 {
		status = consumptionmodels.EntitlementGrantExhausted
	} else if row.ExpiresAt.Valid && !row.ExpiresAt.Time.After(time.Now()) {
		status = consumptionmodels.EntitlementGrantExpired
	}
	return consumptionmodels.EntitlementGrant{
		ID: row.ID, GrantedQuantity: row.GrantedQuantity, RemainingQuantity: row.RemainingQuantity,
		Status: status, ExpiresAt: entitlementTimePointer(row.ExpiresAt), CreatedAt: row.CreatedAt.Time,
	}
}
