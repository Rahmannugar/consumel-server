//go:build integration

package integration_test

import (
	"errors"
	"math"
	"sync"
	"testing"
	"time"

	consumptionmodels "github.com/Rahmannugar/consumel-server/internal/core/consumption/models"
	consumptionrepositories "github.com/Rahmannugar/consumel-server/internal/core/consumption/repositories"
	consumptionservices "github.com/Rahmannugar/consumel-server/internal/core/consumption/services"
	"github.com/Rahmannugar/consumel-server/internal/infra/database/testdb"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRecurringAllowancesRecoverAndSerializeWithConsume(t *testing.T) {
	pool := testdb.OpenMigratedDatabase(t)
	f := createBalanceFixture(t, pool)
	repository := consumptionrepositories.NewAllowanceRepository(pool)
	service := consumptionservices.NewAllowanceService(repository)
	balances := consumptionservices.NewBalanceService(consumptionrepositories.NewBalanceRepository(pool))
	consume := consumptionservices.NewConsumeService(consumptionrepositories.NewConsumeRepository(pool))
	source := consumptionmodels.BalanceMutationSource{Type: consumptionmodels.BalanceSourceDashboardUser, ActorID: allowanceOwner(t, pool)}
	anchor := time.Date(time.Now().Year(), time.Now().Month()-6, 1, 0, 0, 0, 0, time.UTC)
	request := consumptionmodels.AllowanceRequest{StartsAt: anchor, Quantity: 100, RolloverCap: 150, Enabled: true}
	policy, err := service.Save(t.Context(), f.sandboxID, f.customerID, f.meterKey, source, request)
	if err != nil || policy.Version != 1 || policy.Current == nil {
		t.Fatalf("create = %#v, %v", policy, err)
	}
	if _, err = service.Get(t.Context(), f.liveID, f.customerID, f.meterKey); !errors.Is(err, consumptionmodels.ErrAllowanceNotFound) {
		t.Fatalf("environment leaked: %v", err)
	}
	addBalance(t, balances, f.sandboxID, f.customerID, f.meterKey, 7)
	_, _, err = consume.Consume(t.Context(), f.sandboxID, uuid.Nil, newV7(t).String(), consumptionmodels.ConsumeRequest{CustomerID: f.customerID, MeterKey: f.meterKey, Quantity: 40})
	if err != nil {
		t.Fatal(err)
	}
	moveAllowanceIntoPast(t, pool, policy.ID, anchor, 4)
	var balanceID uuid.UUID
	if err = pool.QueryRow(t.Context(), "SELECT balance_id FROM recurring_allowances WHERE id=$1", policy.ID).Scan(&balanceID); err != nil {
		t.Fatal(err)
	}
	const parallel = 8
	results := make(chan error, parallel)
	var group sync.WaitGroup
	for i := range parallel {
		group.Add(1)
		key := newV7(t)
		go func() {
			defer group.Done()
			if i%2 == 0 {
				results <- repository.Renew(t.Context(), balanceID)
			} else {
				_, _, e := consume.Consume(t.Context(), f.sandboxID, uuid.Nil, key.String(), consumptionmodels.ConsumeRequest{CustomerID: f.customerID, MeterKey: f.meterKey, Quantity: 1})
				results <- e
			}
		}()
	}
	group.Wait()
	close(results)
	for err = range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	balance, err := balances.Get(t.Context(), f.sandboxID, f.customerID, f.meterKey)
	if err != nil || balance.Quantity != 253 {
		t.Fatalf("renewed balance = %#v, %v; want 250 policy + 7 manual - 4 consumes", balance, err)
	}
	var periods, outbox int
	if err = pool.QueryRow(t.Context(), "SELECT count(*) FROM allowance_periods WHERE allowance_id=$1", policy.ID).Scan(&periods); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(t.Context(), "SELECT count(*) FROM outbox_events WHERE aggregate_id=$1 AND event_type='allowance.period.recorded.v1'", policy.ID).Scan(&outbox); err != nil {
		t.Fatal(err)
	}
	if periods != 2 || outbox != 2 {
		t.Fatalf("duplicate issuance: periods=%d,outbox=%d", periods, outbox)
	}
	request.Version = policy.Version
	request.Quantity = 50
	request.RolloverCap = 0
	updated, err := service.Save(t.Context(), f.sandboxID, f.customerID, f.meterKey, source, request)
	if err != nil || updated.Current.Quantity != 100 || updated.Pending == nil || updated.Pending.Quantity != 50 {
		t.Fatalf("scheduled edit = %#v, %v", updated, err)
	}
	if _, err = service.Save(t.Context(), f.sandboxID, f.customerID, f.meterKey, source, request); !errors.Is(err, consumptionmodels.ErrAllowanceConflict) {
		t.Fatalf("stale update = %v", err)
	}
	// Advance the pending revision and last period together as a deterministic
	// renewal fixture; the worker and HTTP path use the same reconciliation code.
	moveAllowanceIntoPast(t, pool, policy.ID, anchor, 1)
	start, _, _ := consumptionmodels.AllowancePeriod(anchor, time.Now())
	if _, err = pool.Exec(t.Context(), "UPDATE allowance_revisions SET effective_at=$2 WHERE allowance_id=$1 AND version=2", policy.ID, start); err != nil {
		t.Fatal(err)
	}
	if err = repository.Renew(t.Context(), balanceID); err != nil {
		t.Fatal(err)
	}
	balance, err = balances.Get(t.Context(), f.sandboxID, f.customerID, f.meterKey)
	if err != nil || balance.Quantity != 57 {
		t.Fatalf("reset changed manual grant: %#v, %v", balance, err)
	}
	request.Version = updated.Version
	request.Enabled = false
	stopped, err := service.Save(t.Context(), f.sandboxID, f.customerID, f.meterKey, source, request)
	if err != nil || stopped.Pending == nil || stopped.Current == nil || !stopped.Current.Enabled || stopped.Pending.Enabled {
		t.Fatalf("scheduled stop = %#v,%v", stopped, err)
	}
	moveAllowanceIntoPast(t, pool, policy.ID, anchor, 1)
	if _, err = pool.Exec(t.Context(), "UPDATE allowance_revisions SET effective_at=$2 WHERE allowance_id=$1 AND version=3", policy.ID, start); err != nil {
		t.Fatal(err)
	}
	if err = repository.Renew(t.Context(), balanceID); err != nil {
		t.Fatal(err)
	}
	balance, err = balances.Get(t.Context(), f.sandboxID, f.customerID, f.meterKey)
	if err != nil || balance.Quantity != 7 {
		t.Fatalf("stop changed manual units: %#v,%v", balance, err)
	}
	page, err := service.Periods(t.Context(), f.sandboxID, f.customerID, f.meterKey, nil, 1)
	if err != nil || len(page.Periods) != 1 || page.NextCursor == nil || page.Periods[0].Quantity != 0 {
		t.Fatalf("period page = %#v,%v", page, err)
	}
	cursor, err := consumptionmodels.DecodeOperationCursor(*page.NextCursor)
	if err != nil {
		t.Fatal(err)
	}
	next, err := service.Periods(t.Context(), f.sandboxID, f.customerID, f.meterKey, cursor, 1)
	if err != nil || len(next.Periods) != 1 || !next.Periods[0].PeriodStart.Before(page.Periods[0].PeriodStart) {
		t.Fatalf("next period page = %#v,%v", next, err)
	}
	cursor.ID = uuid.New()
	if _, err = service.Periods(t.Context(), f.sandboxID, f.customerID, f.meterKey, cursor, 1); !errors.Is(err, consumptionmodels.ErrAllowanceCursorInvalid) {
		t.Fatalf("cross-policy cursor = %v", err)
	}
}
func TestRecurringAllowanceRollsBackAndCanRecoverFromOverflow(t *testing.T) {
	pool := testdb.OpenMigratedDatabase(t)
	f := createBalanceFixture(t, pool)
	service := consumptionservices.NewAllowanceService(consumptionrepositories.NewAllowanceRepository(pool))
	balances := consumptionservices.NewBalanceService(consumptionrepositories.NewBalanceRepository(pool))
	source := consumptionmodels.BalanceMutationSource{Type: consumptionmodels.BalanceSourceDashboardUser, ActorID: allowanceOwner(t, pool)}
	anchor := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
	policy, err := service.Save(t.Context(), f.sandboxID, f.customerID, f.meterKey, source, consumptionmodels.AllowanceRequest{StartsAt: anchor, Quantity: 100, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	// Moving the initial period back also makes its grant expired. Manual stock
	// may then fill the balance before a delayed issuance is attempted.
	moveAllowanceIntoPast(t, pool, policy.ID, anchor, 1)
	if _, err = pool.Exec(t.Context(), "UPDATE allowance_revisions SET effective_at=$2 WHERE allowance_id=$1", policy.ID, anchor.AddDate(0, -1, 0)); err != nil {
		t.Fatal(err)
	}
	var balanceID uuid.UUID
	if err = pool.QueryRow(t.Context(), "SELECT balance_id FROM recurring_allowances WHERE id=$1", policy.ID).Scan(&balanceID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(t.Context(), "INSERT INTO entitlement_grants(id,balance_id,source_type,granted_quantity,remaining_quantity) VALUES($1,$2,'manual',$3,$3)", newV7(t), balanceID, int64(math.MaxInt64)); err != nil {
		t.Fatal(err)
	}
	repository := consumptionrepositories.NewAllowanceRepository(pool)
	if err = repository.Renew(t.Context(), balanceID); !errors.Is(err, consumptionmodels.ErrBalanceOverflow) {
		t.Fatalf("overflow error = %v", err)
	}
	var count int
	if err = pool.QueryRow(t.Context(), "SELECT count(*) FROM allowance_periods WHERE allowance_id=$1", policy.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("overflow committed issuance: %d", count)
	}
	balance, err := balances.Set(t.Context(), f.sandboxID, legacyBalanceSource(), f.customerID, f.meterKey, consumptionmodels.SetBalanceRequest{Quantity: 0})
	if err != nil || balance.Quantity != 0 {
		t.Fatalf("overflow recovery = %#v, %v", balance, err)
	}
	// A failing outbox insert must undo the grant, period and schedule together.
	moveAllowanceIntoPast(t, pool, policy.ID, anchor, 1)
	if _, err = pool.Exec(t.Context(), "CREATE FUNCTION reject_allowance_event() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.event_type='allowance.period.recorded.v1' THEN RAISE EXCEPTION 'injected outbox failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER reject_allowance BEFORE INSERT ON outbox_events FOR EACH ROW EXECUTE FUNCTION reject_allowance_event()"); err != nil {
		t.Fatal(err)
	}
	if err = repository.Renew(t.Context(), balanceID); err == nil {
		t.Fatal("outbox failure accepted")
	}
	if _, err = pool.Exec(t.Context(), "DROP TRIGGER reject_allowance ON outbox_events"); err != nil {
		t.Fatal(err)
	}
	if err = repository.Renew(t.Context(), balanceID); err != nil {
		t.Fatalf("retry after rollback: %v", err)
	}
}
func allowanceOwner(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := pool.QueryRow(t.Context(), "SELECT owner_user_id FROM organizations LIMIT 1").Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}
func moveAllowanceIntoPast(t *testing.T, pool *pgxpool.Pool, id uuid.UUID, anchor time.Time, months int) {
	t.Helper()
	start, _, index := consumptionmodels.AllowancePeriod(anchor, time.Now())
	oldStart := consumptionmodels.MonthBoundary(anchor, index-months)
	oldEnd := consumptionmodels.MonthBoundary(anchor, index-months+1)
	// Remove older fixture periods before moving the most recent one; production
	// period identities are immutable and protected by their primary key.
	if _, err := pool.Exec(t.Context(), "DELETE FROM allowance_periods WHERE allowance_id=$1 AND period_start<>$2", id, start); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), "UPDATE entitlement_grants SET expires_at=$2 WHERE id=(SELECT grant_id FROM allowance_periods WHERE allowance_id=$1)", id, oldEnd); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), "UPDATE allowance_periods SET period_start=$2,period_end=$3 WHERE allowance_id=$1", id, oldStart, oldEnd); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), "UPDATE recurring_allowances SET next_period_at=$2 WHERE id=$1", id, oldEnd); err != nil {
		t.Fatal(err)
	}
}
