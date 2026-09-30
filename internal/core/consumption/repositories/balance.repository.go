package repositories

import (
	"context"
	"errors"
	"fmt"

	consumptionmodels "github.com/Rahmannugar/consumel-server/internal/core/consumption/models"
	consumptiondb "github.com/Rahmannugar/consumel-server/internal/core/consumption/repositories/generated"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	request consumptionmodels.AddBalanceRequest,
	idempotencyKey, operationID, balanceID uuid.UUID,
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
		IdempotencyKey:    pgtype.UUID{Bytes: idempotencyKey, Valid: true},
		RequestCustomerID: request.CustomerID, RequestMeterKey: request.MeterKey,
		RequestedQuantity: request.Quantity, CustomerID: subject.CustomerID, MeterID: subject.MeterID,
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

	stored, err := queries.AddBalance(ctx, consumptiondb.AddBalanceParams{
		ID: balanceID, ProjectEnvironmentID: projectEnvironmentID,
		CustomerID: subject.CustomerID, MeterID: subject.MeterID, Quantity: request.Quantity,
	})
	if quantityOverflow(err) {
		return consumptionmodels.Balance{}, false, consumptionmodels.ErrBalanceOverflow
	}
	if err != nil {
		return consumptionmodels.Balance{}, false, fmt.Errorf("add balance: %w", err)
	}
	result := consumptionmodels.Balance{
		ID: stored.ID, ProjectEnvironmentID: projectEnvironmentID,
		CustomerID: request.CustomerID, MeterKey: request.MeterKey,
		Quantity: stored.Quantity, CreatedAt: stored.CreatedAt.Time, UpdatedAt: stored.UpdatedAt.Time,
	}
	if _, err := queries.FinalizeBalanceOperation(ctx, consumptiondb.FinalizeBalanceOperationParams{
		ID: operationID, BalanceID: pgtype.UUID{Bytes: stored.ID, Valid: true},
		ResultingQuantity:  &result.Quantity,
		ResultingCreatedAt: pgtype.Timestamptz{Time: result.CreatedAt, Valid: true},
		ResultingUpdatedAt: pgtype.Timestamptz{Time: result.UpdatedAt, Valid: true},
	}); err != nil {
		return consumptionmodels.Balance{}, false, fmt.Errorf("finalize balance addition: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return consumptionmodels.Balance{}, false, fmt.Errorf("commit balance addition: %w", err)
	}
	return result, false, nil
}

func (repository *BalanceRepository) Set(
	ctx context.Context,
	projectEnvironmentID uuid.UUID,
	customerID, meterKey string,
	quantity int64,
	operationID, balanceID uuid.UUID,
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
	stored, err := queries.SetBalance(ctx, consumptiondb.SetBalanceParams{
		ID: balanceID, ProjectEnvironmentID: projectEnvironmentID,
		CustomerID: subject.CustomerID, MeterID: subject.MeterID, Quantity: quantity,
	})
	if err != nil {
		return consumptionmodels.Balance{}, fmt.Errorf("set exact balance: %w", err)
	}
	result := consumptionmodels.Balance{
		ID: stored.ID, ProjectEnvironmentID: projectEnvironmentID,
		CustomerID: customerID, MeterKey: meterKey, Quantity: stored.Quantity,
		CreatedAt: stored.CreatedAt.Time, UpdatedAt: stored.UpdatedAt.Time,
	}
	if err := queries.RecordBalanceSet(ctx, consumptiondb.RecordBalanceSetParams{
		ID: operationID, ProjectEnvironmentID: projectEnvironmentID,
		RequestCustomerID: customerID, RequestMeterKey: meterKey, RequestedQuantity: quantity,
		CustomerID: subject.CustomerID, MeterID: subject.MeterID,
		BalanceID: pgtype.UUID{Bytes: stored.ID, Valid: true}, ResultingQuantity: &result.Quantity,
		ResultingCreatedAt: pgtype.Timestamptz{Time: result.CreatedAt, Valid: true},
		ResultingUpdatedAt: pgtype.Timestamptz{Time: result.UpdatedAt, Valid: true},
	}); err != nil {
		return consumptionmodels.Balance{}, fmt.Errorf("record exact balance update: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return consumptionmodels.Balance{}, fmt.Errorf("commit exact balance update: %w", err)
	}
	return result, nil
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
		CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
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
			CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
		})
	}
	return balances, nil
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
		operation.RequestedQuantity != request.Quantity {
		return consumptionmodels.Balance{}, consumptionmodels.ErrIdempotencyKeyConflict
	}
	if !operation.BalanceID.Valid || operation.ResultingQuantity == nil ||
		!operation.ResultingCreatedAt.Valid || !operation.ResultingUpdatedAt.Valid {
		return consumptionmodels.Balance{}, fmt.Errorf("balance idempotency result is incomplete")
	}
	return consumptionmodels.Balance{
		ID: operation.BalanceID.Bytes, ProjectEnvironmentID: operation.ProjectEnvironmentID,
		CustomerID: operation.RequestCustomerID, MeterKey: operation.RequestMeterKey,
		Quantity:  *operation.ResultingQuantity,
		CreatedAt: operation.ResultingCreatedAt.Time, UpdatedAt: operation.ResultingUpdatedAt.Time,
	}, nil
}

func quantityOverflow(err error) bool {
	var databaseError *pgconn.PgError
	return errors.As(err, &databaseError) && databaseError.Code == "22003"
}
