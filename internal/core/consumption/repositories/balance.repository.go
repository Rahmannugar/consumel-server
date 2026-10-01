package repositories

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	consumptionmodels "github.com/Rahmannugar/consumel-server/internal/core/consumption/models"
	consumptiondb "github.com/Rahmannugar/consumel-server/internal/core/consumption/repositories/generated"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BalanceRepository struct {
	pool *pgxpool.Pool
}

func NewBalanceRepository(pool *pgxpool.Pool) *BalanceRepository {
	return &BalanceRepository{pool: pool}
}

func (repository *BalanceRepository) Add(
	ctx context.Context,
	projectEnvironmentID uuid.UUID,
	source consumptionmodels.BalanceMutationSource,
	request consumptionmodels.AddBalanceRequest,
	idempotencyKey, operationID, balanceID, grantID uuid.UUID,
) (consumptionmodels.Balance, bool, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return consumptionmodels.Balance{}, false, fmt.Errorf("begin balance addition: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := consumptiondb.New(tx)

	existing, err := operationByIdempotencyKey(ctx, queries, projectEnvironmentID, idempotencyKey)
	if err == nil {
		balance, replayErr := replayBalance(existing, request)
		return balance, true, replayErr
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return consumptionmodels.Balance{}, false, fmt.Errorf("load balance idempotency result: %w", err)
	}

	subject, err := balanceSubject(ctx, queries, projectEnvironmentID, request.CustomerID, request.MeterKey)
	if err != nil {
		return consumptionmodels.Balance{}, false, err
	}
	_, err = queries.ClaimBalanceAddition(ctx, consumptiondb.ClaimBalanceAdditionParams{
		ID: operationID, ProjectEnvironmentID: projectEnvironmentID,
		IdempotencyKey:     pgtype.UUID{Bytes: idempotencyKey, Valid: true},
		RequestCustomerID:  request.CustomerID,
		RequestMeterKey:    request.MeterKey,
		RequestedQuantity:  request.Quantity,
		RequestedExpiresAt: timestamp(request.ExpiresAt),
		CustomerID:         subject.CustomerID,
		MeterID:            subject.MeterID,
		SourceType:         source.Type,
		SourceUserID:       sourceUserID(source),
		SourceApiKeyID:     sourceAPIKeyID(source),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		existing, loadErr := operationByIdempotencyKey(ctx, queries, projectEnvironmentID, idempotencyKey)
		if loadErr != nil {
			return consumptionmodels.Balance{}, false, fmt.Errorf("load concurrent balance idempotency result: %w", loadErr)
		}
		balance, replayErr := replayBalance(existing, request)
		return balance, true, replayErr
	}
	if err != nil {
		return consumptionmodels.Balance{}, false, fmt.Errorf("claim balance addition: %w", err)
	}

	stored, err := queries.EnsureBalance(ctx, consumptiondb.EnsureBalanceParams{
		ID: balanceID, ProjectEnvironmentID: projectEnvironmentID,
		CustomerID: subject.CustomerID, MeterID: subject.MeterID,
	})
	if err != nil {
		return consumptionmodels.Balance{}, false, fmt.Errorf("ensure balance: %w", err)
	}
	if _, err := queries.BalanceForUpdate(ctx, stored.ID); err != nil {
		return consumptionmodels.Balance{}, false, fmt.Errorf("lock balance: %w", err)
	}
	before, err := queries.ActiveEntitlementSummary(ctx, stored.ID)
	if err != nil {
		return consumptionmodels.Balance{}, false, fmt.Errorf("summarize balance: %w", err)
	}
	if before.Quantity > math.MaxInt64-request.Quantity {
		return consumptionmodels.Balance{}, false, consumptionmodels.ErrBalanceOverflow
	}
	if _, err := queries.InsertEntitlementGrant(ctx, consumptiondb.InsertEntitlementGrantParams{
		ID: grantID, BalanceID: stored.ID, GrantedQuantity: request.Quantity,
		ExpiresAt:                   timestamp(request.ExpiresAt),
		CreatedByBalanceOperationID: pgtype.UUID{Bytes: operationID, Valid: true},
	}); err != nil {
		return consumptionmodels.Balance{}, false, fmt.Errorf("insert entitlement grant: %w", err)
	}
	result, err := finalizeBalance(ctx, queries, operationID, stored, request.CustomerID, request.MeterKey)
	if err != nil {
		return consumptionmodels.Balance{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return consumptionmodels.Balance{}, false, fmt.Errorf("commit balance addition: %w", err)
	}
	return result, false, nil
}

func (repository *BalanceRepository) Set(
	ctx context.Context,
	projectEnvironmentID uuid.UUID,
	source consumptionmodels.BalanceMutationSource,
	customerID, meterKey string,
	quantity int64,
	operationID, balanceID, grantID uuid.UUID,
) (consumptionmodels.Balance, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return consumptionmodels.Balance{}, fmt.Errorf("begin exact balance update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := consumptiondb.New(tx)
	subject, err := balanceSubject(ctx, queries, projectEnvironmentID, customerID, meterKey)
	if err != nil {
		return consumptionmodels.Balance{}, err
	}
	stored, err := queries.EnsureBalance(ctx, consumptiondb.EnsureBalanceParams{
		ID: balanceID, ProjectEnvironmentID: projectEnvironmentID,
		CustomerID: subject.CustomerID, MeterID: subject.MeterID,
	})
	if err != nil {
		return consumptionmodels.Balance{}, fmt.Errorf("ensure exact balance: %w", err)
	}
	if _, err := queries.BalanceForUpdate(ctx, stored.ID); err != nil {
		return consumptionmodels.Balance{}, fmt.Errorf("lock exact balance: %w", err)
	}
	if _, err := queries.RecordBalanceSet(ctx, consumptiondb.RecordBalanceSetParams{
		ID: operationID, ProjectEnvironmentID: projectEnvironmentID,
		RequestCustomerID: customerID, RequestMeterKey: meterKey, RequestedQuantity: quantity,
		CustomerID: subject.CustomerID, MeterID: subject.MeterID,
		SourceType: source.Type, SourceUserID: sourceUserID(source), SourceApiKeyID: sourceAPIKeyID(source),
	}); err != nil {
		return consumptionmodels.Balance{}, fmt.Errorf("record exact balance update: %w", err)
	}
	grants, err := queries.SpendableEntitlementGrants(ctx, stored.ID)
	if err != nil {
		return consumptionmodels.Balance{}, fmt.Errorf("load entitlement grants: %w", err)
	}
	current := sumGrantQuantity(grants)
	switch {
	case quantity > current:
		if _, err := queries.InsertEntitlementGrant(ctx, consumptiondb.InsertEntitlementGrantParams{
			ID: grantID, BalanceID: stored.ID, GrantedQuantity: quantity - current,
			CreatedByBalanceOperationID: pgtype.UUID{Bytes: operationID, Valid: true},
		}); err != nil {
			return consumptionmodels.Balance{}, fmt.Errorf("insert exact balance grant: %w", err)
		}
	case quantity < current:
		if err := reduceGrants(ctx, queries, grants, current-quantity, operationID, uuid.Nil); err != nil {
			return consumptionmodels.Balance{}, err
		}
	}
	result, err := finalizeBalance(ctx, queries, operationID, stored, customerID, meterKey)
	if err != nil {
		return consumptionmodels.Balance{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return consumptionmodels.Balance{}, fmt.Errorf("commit exact balance update: %w", err)
	}
	return result, nil
}

func sourceUserID(source consumptionmodels.BalanceMutationSource) pgtype.UUID {
	return pgtype.UUID{Bytes: source.ActorID, Valid: source.Type == consumptionmodels.BalanceSourceDashboardUser}
}

func sourceAPIKeyID(source consumptionmodels.BalanceMutationSource) pgtype.UUID {
	return pgtype.UUID{Bytes: source.ActorID, Valid: source.Type == consumptionmodels.BalanceSourceAPIKey}
}

func (repository *BalanceRepository) Get(
	ctx context.Context,
	projectEnvironmentID uuid.UUID,
	customerID, meterKey string,
) (consumptionmodels.Balance, error) {
	queries := consumptiondb.New(repository.pool)
	if _, err := balanceSubject(ctx, queries, projectEnvironmentID, customerID, meterKey); err != nil {
		return consumptionmodels.Balance{}, err
	}
	row, err := queries.BalanceByPublicKeys(ctx, consumptiondb.BalanceByPublicKeysParams{
		ProjectEnvironmentID: projectEnvironmentID, CustomerID: customerID, MeterKey: meterKey,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return consumptionmodels.Balance{}, consumptionmodels.ErrBalanceNotFound
	}
	if err != nil {
		return consumptionmodels.Balance{}, fmt.Errorf("get balance: %w", err)
	}
	return consumptionmodels.Balance{
		ID: row.ID, ProjectEnvironmentID: row.ProjectEnvironmentID,
		CustomerID: row.PublicCustomerID, MeterKey: row.MeterKey, Quantity: row.Quantity,
		NextExpiresAt: entitlementTimePointer(row.NextExpiresAt),
		CreatedAt:     row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
	}, nil
}

func (repository *BalanceRepository) List(
	ctx context.Context,
	projectEnvironmentID uuid.UUID,
	customerID string,
) ([]consumptionmodels.Balance, error) {
	queries := consumptiondb.New(repository.pool)
	if _, err := queries.ConsumptionCustomerByPublicID(ctx, consumptiondb.ConsumptionCustomerByPublicIDParams{
		ProjectEnvironmentID: projectEnvironmentID, CustomerID: customerID,
	}); errors.Is(err, pgx.ErrNoRows) {
		return nil, consumptionmodels.ErrBalanceSubjectNotFound
	} else if err != nil {
		return nil, fmt.Errorf("resolve balance customer: %w", err)
	}
	rows, err := queries.ListCustomerBalances(ctx, consumptiondb.ListCustomerBalancesParams{
		ProjectEnvironmentID: projectEnvironmentID, CustomerID: customerID,
	})
	if err != nil {
		return nil, fmt.Errorf("list customer balances: %w", err)
	}
	balances := make([]consumptionmodels.Balance, 0, len(rows))
	for _, row := range rows {
		balances = append(balances, consumptionmodels.Balance{
			ID: row.ID, ProjectEnvironmentID: row.ProjectEnvironmentID,
			CustomerID: row.PublicCustomerID, MeterKey: row.MeterKey, Quantity: row.Quantity,
			NextExpiresAt: entitlementTimePointer(row.NextExpiresAt),
			CreatedAt:     row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
		})
	}
	return balances, nil
}

func finalizeBalance(
	ctx context.Context,
	queries *consumptiondb.Queries,
	operationID uuid.UUID,
	stored consumptiondb.Balance,
	customerID, meterKey string,
) (consumptionmodels.Balance, error) {
	refreshed, err := queries.RefreshBalanceProjection(ctx, stored.ID)
	if err != nil {
		return consumptionmodels.Balance{}, fmt.Errorf("refresh balance projection: %w", err)
	}
	summary, err := queries.ActiveEntitlementSummary(ctx, stored.ID)
	if err != nil {
		return consumptionmodels.Balance{}, fmt.Errorf("summarize updated balance: %w", err)
	}
	result := consumptionmodels.Balance{
		ID: refreshed.ID, ProjectEnvironmentID: refreshed.ProjectEnvironmentID,
		CustomerID: customerID, MeterKey: meterKey, Quantity: summary.Quantity,
		NextExpiresAt: entitlementTimePointer(summary.NextExpiresAt),
		CreatedAt:     refreshed.CreatedAt.Time, UpdatedAt: refreshed.UpdatedAt.Time,
	}
	if _, err := queries.FinalizeBalanceOperation(ctx, consumptiondb.FinalizeBalanceOperationParams{
		ID: operationID, BalanceID: pgtype.UUID{Bytes: refreshed.ID, Valid: true},
		ResultingQuantity:      &result.Quantity,
		ResultingNextExpiresAt: timestamp(result.NextExpiresAt),
		ResultingCreatedAt:     pgtype.Timestamptz{Time: result.CreatedAt, Valid: true},
		ResultingUpdatedAt:     pgtype.Timestamptz{Time: result.UpdatedAt, Valid: true},
	}); err != nil {
		return consumptionmodels.Balance{}, fmt.Errorf("finalize balance operation: %w", err)
	}
	return result, nil
}

func reduceGrants(
	ctx context.Context,
	queries *consumptiondb.Queries,
	grants []consumptiondb.EntitlementGrant,
	quantity int64,
	balanceOperationID, consumptionOperationID uuid.UUID,
) error {
	remaining := quantity
	for _, grant := range grants {
		if remaining == 0 {
			break
		}
		allocated := min(grant.RemainingQuantity, remaining)
		if err := queries.UpdateEntitlementGrantRemaining(ctx, consumptiondb.UpdateEntitlementGrantRemainingParams{
			ID: grant.ID, RemainingQuantity: grant.RemainingQuantity - allocated,
		}); err != nil {
			return fmt.Errorf("debit entitlement grant: %w", err)
		}
		if balanceOperationID != uuid.Nil {
			if err := queries.RecordBalanceGrantAllocation(ctx, consumptiondb.RecordBalanceGrantAllocationParams{
				BalanceOperationID: balanceOperationID, EntitlementGrantID: grant.ID, Quantity: allocated,
			}); err != nil {
				return fmt.Errorf("record balance grant allocation: %w", err)
			}
		}
		if consumptionOperationID != uuid.Nil {
			if err := queries.RecordConsumptionGrantAllocation(ctx, consumptiondb.RecordConsumptionGrantAllocationParams{
				ConsumptionOperationID: consumptionOperationID, EntitlementGrantID: grant.ID, Quantity: allocated,
			}); err != nil {
				return fmt.Errorf("record consumption grant allocation: %w", err)
			}
		}
		remaining -= allocated
	}
	if remaining != 0 {
		return fmt.Errorf("entitlement grant allocation is incomplete")
	}
	return nil
}

func sumGrantQuantity(grants []consumptiondb.EntitlementGrant) int64 {
	var total int64
	for _, grant := range grants {
		total += grant.RemainingQuantity
	}
	return total
}

type balanceQueries interface {
	BalanceOperationByIdempotencyKey(context.Context, consumptiondb.BalanceOperationByIdempotencyKeyParams) (consumptiondb.BalanceOperation, error)
	BalanceSubjectByPublicKeys(context.Context, consumptiondb.BalanceSubjectByPublicKeysParams) (consumptiondb.BalanceSubjectByPublicKeysRow, error)
}

func operationByIdempotencyKey(
	ctx context.Context,
	queries balanceQueries,
	projectEnvironmentID, idempotencyKey uuid.UUID,
) (consumptiondb.BalanceOperation, error) {
	return queries.BalanceOperationByIdempotencyKey(ctx, consumptiondb.BalanceOperationByIdempotencyKeyParams{
		ProjectEnvironmentID: projectEnvironmentID,
		IdempotencyKey:       pgtype.UUID{Bytes: idempotencyKey, Valid: true},
	})
}

func balanceSubject(
	ctx context.Context,
	queries balanceQueries,
	projectEnvironmentID uuid.UUID,
	customerID, meterKey string,
) (consumptionmodels.Subject, error) {
	row, err := queries.BalanceSubjectByPublicKeys(ctx, consumptiondb.BalanceSubjectByPublicKeysParams{
		ProjectEnvironmentID: projectEnvironmentID, CustomerID: customerID, MeterKey: meterKey,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return consumptionmodels.Subject{}, consumptionmodels.ErrBalanceSubjectNotFound
	}
	if err != nil {
		return consumptionmodels.Subject{}, fmt.Errorf("resolve balance customer and meter: %w", err)
	}
	return consumptionmodels.Subject{CustomerID: row.CustomerID, MeterID: row.MeterID}, nil
}

func replayBalance(
	operation consumptiondb.BalanceOperation,
	request consumptionmodels.AddBalanceRequest,
) (consumptionmodels.Balance, error) {
	if operation.RequestCustomerID != request.CustomerID ||
		operation.RequestMeterKey != request.MeterKey ||
		operation.RequestedQuantity != request.Quantity ||
		!sameTimestamp(operation.RequestedExpiresAt, request.ExpiresAt) {
		return consumptionmodels.Balance{}, consumptionmodels.ErrIdempotencyKeyConflict
	}
	if !operation.BalanceID.Valid || operation.ResultingQuantity == nil ||
		!operation.ResultingCreatedAt.Valid || !operation.ResultingUpdatedAt.Valid {
		return consumptionmodels.Balance{}, fmt.Errorf("balance idempotency result is incomplete")
	}
	return consumptionmodels.Balance{
		ID: operation.BalanceID.Bytes, ProjectEnvironmentID: operation.ProjectEnvironmentID,
		CustomerID: operation.RequestCustomerID, MeterKey: operation.RequestMeterKey,
		Quantity:      *operation.ResultingQuantity,
		NextExpiresAt: entitlementTimePointer(operation.ResultingNextExpiresAt),
		CreatedAt:     operation.ResultingCreatedAt.Time, UpdatedAt: operation.ResultingUpdatedAt.Time,
	}, nil
}

func timestamp(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: value.UTC(), Valid: true}
}

func entitlementTimePointer(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}

func sameTimestamp(stored pgtype.Timestamptz, requested *time.Time) bool {
	if requested == nil {
		return !stored.Valid
	}
	return stored.Valid && stored.Time.Equal(*requested)
}
