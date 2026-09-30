package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	consumptionmodels "github.com/Rahmannugar/consumel-server/internal/core/consumption/models"
	consumptiondb "github.com/Rahmannugar/consumel-server/internal/core/consumption/repositories/generated"
	"github.com/Rahmannugar/consumel-server/internal/infra/events"
	metermodels "github.com/Rahmannugar/consumel-server/internal/meters/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ConsumeRepository struct {
	pool *pgxpool.Pool
}

func NewConsumeRepository(pool *pgxpool.Pool) *ConsumeRepository {
	return &ConsumeRepository{pool: pool}
}

func (repository *ConsumeRepository) Consume(
	ctx context.Context,
	projectEnvironmentID uuid.UUID,
	request consumptionmodels.ConsumeRequest,
	idempotencyKey, operationID, customerID, balanceID, outboxID uuid.UUID,
) (consumptionmodels.UsageEvent, bool, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return consumptionmodels.UsageEvent{}, false, fmt.Errorf("begin consumption: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := consumptiondb.New(tx)

	existing, err := consumptionOperation(ctx, queries, projectEnvironmentID, idempotencyKey)
	if err == nil {
		return recordConsumptionReplay(ctx, tx, queries, existing, request)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return consumptionmodels.UsageEvent{}, false, fmt.Errorf("load consumption idempotency result: %w", err)
	}

	meter, err := queries.ActiveConsumptionMeter(ctx, consumptiondb.ActiveConsumptionMeterParams{
		ProjectEnvironmentID: projectEnvironmentID,
		MeterKey:             request.MeterKey,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return consumptionmodels.UsageEvent{}, false, consumptionmodels.ErrConsumeMeterNotFound
	}
	if err != nil {
		return consumptionmodels.UsageEvent{}, false, fmt.Errorf("resolve consumption meter: %w", err)
	}
	resolvedCustomerID, err := queries.EnsureConsumptionCustomer(ctx, consumptiondb.EnsureConsumptionCustomerParams{
		ID: customerID, ProjectEnvironmentID: projectEnvironmentID, CustomerID: request.CustomerID,
	})
	if err != nil {
		return consumptionmodels.UsageEvent{}, false, fmt.Errorf("resolve consumption customer: %w", err)
	}
	claimed, err := queries.ClaimConsumptionOperation(ctx, consumptiondb.ClaimConsumptionOperationParams{
		ID: operationID, ProjectEnvironmentID: projectEnvironmentID, IdempotencyKey: idempotencyKey,
		RequestCustomerID: request.CustomerID, RequestMeterKey: request.MeterKey,
		RequestedQuantity: request.Quantity, CustomerID: resolvedCustomerID,
		MeterID: meter.ID, MeterType: meter.MeterType,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		existing, loadErr := consumptionOperation(ctx, queries, projectEnvironmentID, idempotencyKey)
		if loadErr != nil {
			return consumptionmodels.UsageEvent{}, false,
				fmt.Errorf("load concurrent consumption idempotency result: %w", loadErr)
		}
		return recordConsumptionReplay(ctx, tx, queries, existing, request)
	}
	if err != nil {
		return consumptionmodels.UsageEvent{}, false, fmt.Errorf("claim consumption: %w", err)
	}

	result, err := repository.applyConsumption(
		ctx, queries, claimed, request, balanceID, meter.EnvironmentName == "live",
	)
	if errors.Is(err, consumptionmodels.ErrInsufficientBalance) {
		if outboxErr := insertConsumptionOutbox(
			ctx, tx, outboxID, claimed.ID, projectEnvironmentID, "usage.denied.v1",
			claimed.CreatedAt.Time,
		); outboxErr != nil {
			return consumptionmodels.UsageEvent{}, false, outboxErr
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			return consumptionmodels.UsageEvent{}, false, fmt.Errorf("commit denied consumption: %w", commitErr)
		}
		return consumptionmodels.UsageEvent{}, false, err
	}
	if err != nil {
		return consumptionmodels.UsageEvent{}, false, err
	}
	if err := insertConsumptionOutbox(
		ctx, tx, outboxID, result.ID, projectEnvironmentID, "usage.consumed.v1",
		result.CreatedAt,
	); err != nil {
		return consumptionmodels.UsageEvent{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return consumptionmodels.UsageEvent{}, false, fmt.Errorf("commit consumption: %w", err)
	}
	return result, false, nil
}

func recordConsumptionReplay(
	ctx context.Context,
	tx pgx.Tx,
	queries *consumptiondb.Queries,
	operation consumptiondb.ConsumptionOperation,
	request consumptionmodels.ConsumeRequest,
) (consumptionmodels.UsageEvent, bool, error) {
	_, replayErr := replayConsumption(operation, request)
	if errors.Is(replayErr, consumptionmodels.ErrIdempotencyKeyConflict) {
		return consumptionmodels.UsageEvent{}, true, replayErr
	}
	if replayErr != nil && !errors.Is(replayErr, consumptionmodels.ErrInsufficientBalance) {
		return consumptionmodels.UsageEvent{}, true, replayErr
	}
	updated, err := queries.RecordConsumptionReplay(ctx, operation.ID)
	if err != nil {
		return consumptionmodels.UsageEvent{}, true, fmt.Errorf("record consumption replay: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return consumptionmodels.UsageEvent{}, true, fmt.Errorf("commit consumption replay: %w", err)
	}
	if replayErr != nil {
		return consumptionmodels.UsageEvent{}, true, replayErr
	}
	return usageEvent(updated), true, nil
}

func insertConsumptionOutbox(
	ctx context.Context,
	tx pgx.Tx,
	outboxID, operationID, projectEnvironmentID uuid.UUID,
	eventType string,
	occurredAt time.Time,
) error {
	payload, err := json.Marshal(map[string]string{
		"usageEventId":         operationID.String(),
		"projectEnvironmentId": projectEnvironmentID.String(),
	})
	if err != nil {
		return fmt.Errorf("encode consumption outbox event: %w", err)
	}
	if err := events.Insert(ctx, tx, events.Event{
		ID: outboxID, Type: eventType, AggregateType: "usage_event",
		AggregateID: operationID, Payload: payload, OccurredAt: occurredAt,
	}); err != nil {
		return err
	}
	return nil
}

func (repository *ConsumeRepository) applyConsumption(
	ctx context.Context,
	queries *consumptiondb.Queries,
	operation consumptiondb.ConsumptionOperation,
	request consumptionmodels.ConsumeRequest,
	balanceID uuid.UUID,
	billable bool,
) (consumptionmodels.UsageEvent, error) {
	meterType := metermodels.MeterType(operation.MeterType)
	var balance consumptiondb.Balance
	var remaining *int64
	var debited int64

	switch meterType {
	case metermodels.MeterTypePrepaid:
		stored, err := lockedBalance(ctx, queries, operation)
		if errors.Is(err, pgx.ErrNoRows) {
			return consumptionmodels.UsageEvent{}, denyConsumption(ctx, queries, operation, nil, 0)
		}
		if err != nil {
			return consumptionmodels.UsageEvent{}, fmt.Errorf("lock prepaid balance: %w", err)
		}
		grants, err := queries.SpendableEntitlementGrants(ctx, stored.ID)
		if err != nil {
			return consumptionmodels.UsageEvent{}, fmt.Errorf("load prepaid entitlement grants: %w", err)
		}
		available := sumGrantQuantity(grants)
		if available < request.Quantity {
			return consumptionmodels.UsageEvent{}, denyConsumption(ctx, queries, operation, &stored, available)
		}
		err = reduceGrants(ctx, queries, grants, request.Quantity, uuid.Nil, operation.ID)
		if err != nil {
			return consumptionmodels.UsageEvent{}, fmt.Errorf("debit prepaid entitlement grants: %w", err)
		}
		balance, err = refreshConsumedBalance(ctx, queries, stored)
		if err != nil {
			return consumptionmodels.UsageEvent{}, err
		}
		debited = request.Quantity
		remaining = &balance.Quantity
	case metermodels.MeterTypeHybrid:
		if err := queries.EnsureHybridBalance(ctx, consumptiondb.EnsureHybridBalanceParams{
			ID: balanceID, ProjectEnvironmentID: operation.ProjectEnvironmentID,
			CustomerID: operation.CustomerID, MeterID: operation.MeterID,
		}); err != nil {
			return consumptionmodels.UsageEvent{}, fmt.Errorf("ensure hybrid balance: %w", err)
		}
		stored, err := lockedBalance(ctx, queries, operation)
		if err != nil {
			return consumptionmodels.UsageEvent{}, fmt.Errorf("lock hybrid balance: %w", err)
		}
		grants, err := queries.SpendableEntitlementGrants(ctx, stored.ID)
		if err != nil {
			return consumptionmodels.UsageEvent{}, fmt.Errorf("load hybrid entitlement grants: %w", err)
		}
		debited = min(sumGrantQuantity(grants), request.Quantity)
		err = reduceGrants(ctx, queries, grants, debited, uuid.Nil, operation.ID)
		if err != nil {
			return consumptionmodels.UsageEvent{}, fmt.Errorf("debit hybrid entitlement grants: %w", err)
		}
		balance, err = refreshConsumedBalance(ctx, queries, stored)
		if err != nil {
			return consumptionmodels.UsageEvent{}, err
		}
		remaining = &balance.Quantity
	case metermodels.MeterTypePostpaid:
		// Postpaid usage is accepted without coupling the synchronous decision to a balance.
	default:
		return consumptionmodels.UsageEvent{}, fmt.Errorf("unsupported consumption meter type %q", meterType)
	}

	var storedBalanceID pgtype.UUID
	if remaining != nil {
		storedBalanceID = pgtype.UUID{Bytes: balance.ID, Valid: true}
	}
	accepted, err := queries.AcceptConsumptionOperation(ctx, consumptiondb.AcceptConsumptionOperationParams{
		ID: operation.ID, BalanceID: storedBalanceID, BalanceDebited: debited,
		ResultingBalance: remaining, Billable: billable,
	})
	if err != nil {
		return consumptionmodels.UsageEvent{}, fmt.Errorf("accept consumption: %w", err)
	}
	return usageEvent(accepted), nil
}

func lockedBalance(
	ctx context.Context,
	queries *consumptiondb.Queries,
	operation consumptiondb.ConsumptionOperation,
) (consumptiondb.Balance, error) {
	return queries.BalanceForConsumption(ctx, consumptiondb.BalanceForConsumptionParams{
		ProjectEnvironmentID: operation.ProjectEnvironmentID,
		CustomerID:           operation.CustomerID,
		MeterID:              operation.MeterID,
	})
}

func refreshConsumedBalance(
	ctx context.Context,
	queries *consumptiondb.Queries,
	balance consumptiondb.Balance,
) (consumptiondb.Balance, error) {
	updated, err := queries.RefreshBalanceProjection(ctx, balance.ID)
	if err != nil {
		return consumptiondb.Balance{}, fmt.Errorf("refresh consumed balance: %w", err)
	}
	return updated, nil
}

func denyConsumption(
	ctx context.Context,
	queries *consumptiondb.Queries,
	operation consumptiondb.ConsumptionOperation,
	balance *consumptiondb.Balance,
	available int64,
) error {
	var balanceID pgtype.UUID
	var remaining *int64
	if balance != nil {
		balanceID = pgtype.UUID{Bytes: balance.ID, Valid: true}
		remaining = &available
	}
	if _, err := queries.DenyConsumptionOperation(ctx, consumptiondb.DenyConsumptionOperationParams{
		ID: operation.ID, BalanceID: balanceID, ResultingBalance: remaining,
	}); err != nil {
		return fmt.Errorf("deny consumption: %w", err)
	}
	return consumptionmodels.ErrInsufficientBalance
}

func consumptionOperation(
	ctx context.Context,
	queries *consumptiondb.Queries,
	projectEnvironmentID, idempotencyKey uuid.UUID,
) (consumptiondb.ConsumptionOperation, error) {
	return queries.ConsumptionOperationByIdempotencyKey(
		ctx,
		consumptiondb.ConsumptionOperationByIdempotencyKeyParams{
			ProjectEnvironmentID: projectEnvironmentID, IdempotencyKey: idempotencyKey,
		},
	)
}

func replayConsumption(
	operation consumptiondb.ConsumptionOperation,
	request consumptionmodels.ConsumeRequest,
) (consumptionmodels.UsageEvent, error) {
	if operation.RequestCustomerID != request.CustomerID ||
		operation.RequestMeterKey != request.MeterKey ||
		operation.RequestedQuantity != request.Quantity {
		return consumptionmodels.UsageEvent{}, consumptionmodels.ErrIdempotencyKeyConflict
	}
	switch operation.Status {
	case "accepted":
		return usageEvent(operation), nil
	case "denied":
		return consumptionmodels.UsageEvent{}, consumptionmodels.ErrInsufficientBalance
	default:
		return consumptionmodels.UsageEvent{}, fmt.Errorf("consumption idempotency result is incomplete")
	}
}

func usageEvent(operation consumptiondb.ConsumptionOperation) consumptionmodels.UsageEvent {
	return consumptionmodels.UsageEvent{
		ID: operation.ID, ProjectEnvironmentID: operation.ProjectEnvironmentID,
		CustomerID: operation.RequestCustomerID, MeterKey: operation.RequestMeterKey,
		Quantity: operation.RequestedQuantity, MeterType: metermodels.MeterType(operation.MeterType),
		BalanceDebited: operation.BalanceDebited, RemainingBalance: operation.ResultingBalance,
		Billable: operation.Billable, CreatedAt: operation.CreatedAt.Time,
	}
}
