//go:build integration

package integration_test

import (
	"errors"
	"sync"
	"testing"

	consumptionmodels "github.com/Rahmannugar/consumel-server/internal/core/consumption/models"
	consumptionrepositories "github.com/Rahmannugar/consumel-server/internal/core/consumption/repositories"
	consumptionservices "github.com/Rahmannugar/consumel-server/internal/core/consumption/services"
	"github.com/Rahmannugar/consumel-server/internal/infra/database/testdb"
	metermodels "github.com/Rahmannugar/consumel-server/internal/meters/models"
	meterrepositories "github.com/Rahmannugar/consumel-server/internal/meters/repositories"
	meterservices "github.com/Rahmannugar/consumel-server/internal/meters/services"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestConsumeAppliesMeterBehaviorAndDurableIdempotency(t *testing.T) {
	pool := openConsumptionDatabase(t)
	fixture := createBalanceFixture(t, pool)
	addConsumptionMeters(t, pool, fixture.sandboxID)
	balanceService := consumptionservices.NewBalanceService(consumptionrepositories.NewBalanceRepository(pool))
	consumeService := consumptionservices.NewConsumeService(consumptionrepositories.NewConsumeRepository(pool))
	operationService := consumptionservices.NewOperationService(consumptionrepositories.NewOperationRepository(pool), nil)
	addBalance(t, balanceService, fixture.sandboxID, fixture.customerID, fixture.meterKey, 10)

	key := newV7(t)
	request := consumptionmodels.ConsumeRequest{
		CustomerID: fixture.customerID, MeterKey: fixture.meterKey, Quantity: 4,
	}
	first, replayed, err := consumeService.Consume(t.Context(), fixture.sandboxID, key.String(), request)
	if err != nil || replayed {
		t.Fatalf("prepaid consume = %#v, replayed %t, error %v", first, replayed, err)
	}
	if first.BalanceDebited != 4 || first.RemainingBalance == nil || *first.RemainingBalance != 6 || first.Billable {
		t.Fatalf("prepaid result = %#v", first)
	}
	replay, replayed, err := consumeService.Consume(t.Context(), fixture.sandboxID, key.String(), request)
	if err != nil || !replayed || replay.ID != first.ID {
		t.Fatalf("consume replay = %#v, replayed %t, error %v", replay, replayed, err)
	}
	if _, _, err := consumeService.Consume(t.Context(), fixture.sandboxID, key.String(), consumptionmodels.ConsumeRequest{
		CustomerID: fixture.customerID, MeterKey: fixture.meterKey, Quantity: 3,
	}); !errors.Is(err, consumptionmodels.ErrIdempotencyKeyConflict) {
		t.Fatalf("changed consume replay error = %v", err)
	}

	deniedKey := newV7(t)
	if _, replayed, err := consumeService.Consume(t.Context(), fixture.sandboxID, deniedKey.String(), consumptionmodels.ConsumeRequest{
		CustomerID: fixture.customerID, MeterKey: fixture.meterKey, Quantity: 7,
	}); !errors.Is(err, consumptionmodels.ErrInsufficientBalance) || replayed {
		t.Fatalf("first denied consume replayed %t, error %v", replayed, err)
	}
	if _, replayed, err := consumeService.Consume(t.Context(), fixture.sandboxID, deniedKey.String(), consumptionmodels.ConsumeRequest{
		CustomerID: fixture.customerID, MeterKey: fixture.meterKey, Quantity: 7,
	}); !errors.Is(err, consumptionmodels.ErrInsufficientBalance) || !replayed {
		t.Fatalf("denied replay replayed %t, error %v", replayed, err)
	}

	postpaid, replayed, err := consumeService.Consume(t.Context(), fixture.sandboxID, newV7(t).String(), consumptionmodels.ConsumeRequest{
		CustomerID: "customer_created_by_usage", MeterKey: "bandwidth_gb", Quantity: 12,
	})
	if err != nil || replayed || postpaid.RemainingBalance != nil || postpaid.BalanceDebited != 0 {
		t.Fatalf("postpaid consume = %#v, replayed %t, error %v", postpaid, replayed, err)
	}
	addBalance(t, balanceService, fixture.sandboxID, fixture.customerID, "seats", 3)
	hybrid, _, err := consumeService.Consume(t.Context(), fixture.sandboxID, newV7(t).String(), consumptionmodels.ConsumeRequest{
		CustomerID: fixture.customerID, MeterKey: "seats", Quantity: 5,
	})
	if err != nil || hybrid.BalanceDebited != 3 || hybrid.RemainingBalance == nil || *hybrid.RemainingBalance != 0 {
		t.Fatalf("hybrid consume = %#v, error %v", hybrid, err)
	}

	addBalance(t, balanceService, fixture.liveID, fixture.customerID, fixture.meterKey, 2)
	live, _, err := consumeService.Consume(t.Context(), fixture.liveID, newV7(t).String(), consumptionmodels.ConsumeRequest{
		CustomerID: fixture.customerID, MeterKey: fixture.meterKey, Quantity: 1,
	})
	if err != nil || !live.Billable {
		t.Fatalf("Live consume = %#v, error %v", live, err)
	}

	var accepted, denied, outbox, createdCustomer, replayedOperations int
	if err := pool.QueryRow(t.Context(), `SELECT
		count(*) FILTER (WHERE status = 'accepted'),
		count(*) FILTER (WHERE status = 'denied')
		FROM consumption_operations WHERE project_environment_id = $1`, fixture.sandboxID).Scan(&accepted, &denied); err != nil {
		t.Fatalf("count consumption operations: %v", err)
	}
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM outbox_events WHERE event_type IN ('usage.consumed.v1', 'usage.denied.v1')`).Scan(&outbox); err != nil {
		t.Fatalf("count consumption outbox events: %v", err)
	}
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM customers WHERE project_environment_id = $1 AND customer_id = 'customer_created_by_usage'`, fixture.sandboxID).Scan(&createdCustomer); err != nil {
		t.Fatalf("count usage-created customer: %v", err)
	}
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM consumption_operations WHERE project_environment_id = $1 AND replay_count = 1`, fixture.sandboxID).Scan(&replayedOperations); err != nil {
		t.Fatalf("count replayed consumption operations: %v", err)
	}
	operations, next, err := operationService.List(t.Context(), fixture.sandboxID, nil, 10, "", "", "")
	if err != nil || len(operations) != 4 || next != nil {
		t.Fatalf("listed operations = %#v, next %v, error %v", operations, next, err)
	}
	deniedOperations, _, err := operationService.List(
		t.Context(), fixture.sandboxID, nil, 10, consumptionmodels.OperationStatusDenied, "", "",
	)
	if err != nil || len(deniedOperations) != 1 || deniedOperations[0].DenialReason == nil {
		t.Fatalf("denied operations = %#v, error %v", deniedOperations, err)
	}
	if accepted != 3 || denied != 1 || outbox != 5 || createdCustomer != 1 || replayedOperations != 2 {
		t.Fatalf("accepted %d, denied %d, outbox %d, created customer %d, replayed operations %d", accepted, denied, outbox, createdCustomer, replayedOperations)
	}
}

func TestConcurrentConsumeCannotOverspendOrRepeatOneLogicalOperation(t *testing.T) {
	pool := openConsumptionDatabase(t)
	fixture := createBalanceFixture(t, pool)
	balanceService := consumptionservices.NewBalanceService(consumptionrepositories.NewBalanceRepository(pool))
	consumeService := consumptionservices.NewConsumeService(consumptionrepositories.NewConsumeRepository(pool))
	addBalance(t, balanceService, fixture.sandboxID, fixture.customerID, fixture.meterKey, 10)

	results := make(chan error, 2)
	var concurrent sync.WaitGroup
	for range 2 {
		concurrent.Add(1)
		go func(key uuid.UUID) {
			defer concurrent.Done()
			_, _, err := consumeService.Consume(t.Context(), fixture.sandboxID, key.String(), consumptionmodels.ConsumeRequest{
				CustomerID: fixture.customerID, MeterKey: fixture.meterKey, Quantity: 7,
			})
			results <- err
		}(newV7(t))
	}
	concurrent.Wait()
	close(results)
	accepted, denied := 0, 0
	for err := range results {
		switch {
		case err == nil:
			accepted++
		case errors.Is(err, consumptionmodels.ErrInsufficientBalance):
			denied++
		default:
			t.Fatalf("concurrent consume error = %v", err)
		}
	}
	if accepted != 1 || denied != 1 {
		t.Fatalf("accepted %d and denied %d, want one each", accepted, denied)
	}
	balance, err := balanceService.Get(t.Context(), fixture.sandboxID, fixture.customerID, fixture.meterKey)
	if err != nil || balance.Quantity != 3 {
		t.Fatalf("post-concurrency balance = %#v, error %v", balance, err)
	}

	if _, err := balanceService.Set(t.Context(), fixture.sandboxID, fixture.customerID, fixture.meterKey, consumptionmodels.SetBalanceRequest{Quantity: 3}); err != nil {
		t.Fatalf("reset balance: %v", err)
	}
	const attempts = 8
	key := newV7(t)
	replays := make(chan bool, attempts)
	errorsByAttempt := make(chan error, attempts)
	var retries sync.WaitGroup
	for range attempts {
		retries.Add(1)
		go func() {
			defer retries.Done()
			_, replayed, consumeErr := consumeService.Consume(t.Context(), fixture.sandboxID, key.String(), consumptionmodels.ConsumeRequest{
				CustomerID: fixture.customerID, MeterKey: fixture.meterKey, Quantity: 1,
			})
			replays <- replayed
			errorsByAttempt <- consumeErr
		}()
	}
	retries.Wait()
	close(replays)
	close(errorsByAttempt)
	for consumeErr := range errorsByAttempt {
		if consumeErr != nil {
			t.Fatalf("same-key concurrent consume: %v", consumeErr)
		}
	}
	replayCount := 0
	for replayed := range replays {
		if replayed {
			replayCount++
		}
	}
	if replayCount != attempts-1 {
		t.Fatalf("replay count = %d, want %d", replayCount, attempts-1)
	}
	var storedReplayCount int64
	if err := pool.QueryRow(t.Context(), `SELECT replay_count FROM consumption_operations WHERE project_environment_id = $1 AND idempotency_key = $2`, fixture.sandboxID, key).Scan(&storedReplayCount); err != nil {
		t.Fatalf("load replay count: %v", err)
	}
	if storedReplayCount != attempts-1 {
		t.Fatalf("stored replay count = %d, want %d", storedReplayCount, attempts-1)
	}
	balance, err = balanceService.Get(t.Context(), fixture.sandboxID, fixture.customerID, fixture.meterKey)
	if err != nil || balance.Quantity != 2 {
		t.Fatalf("same-key balance = %#v, error %v", balance, err)
	}
}

func openConsumptionDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return testdb.OpenMigratedDatabase(t)
}

func addConsumptionMeters(t *testing.T, pool *pgxpool.Pool, environmentID uuid.UUID) {
	t.Helper()
	service := meterservices.NewMeterService(meterrepositories.NewMeterRepository(pool))
	for _, request := range []metermodels.CreateMeterRequest{
		{MeterKey: "bandwidth_gb", Name: "Bandwidth", Type: metermodels.MeterTypePostpaid},
		{MeterKey: "seats", Name: "Seats", Type: metermodels.MeterTypeHybrid},
	} {
		if _, err := service.Create(t.Context(), environmentID, request); err != nil {
			t.Fatalf("create %s meter: %v", request.MeterKey, err)
		}
	}
}

func addBalance(
	t *testing.T,
	service *consumptionservices.BalanceService,
	environmentID uuid.UUID,
	customerID, meterKey string,
	quantity int64,
) {
	t.Helper()
	if _, _, err := service.Add(t.Context(), environmentID, newV7(t).String(), consumptionmodels.AddBalanceRequest{
		CustomerID: customerID, MeterKey: meterKey, Quantity: quantity,
	}); err != nil {
		t.Fatalf("add %s balance: %v", meterKey, err)
	}
}
