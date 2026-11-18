package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/Rahmannugar/consumel-server/internal/common/ids"
	consumptionmodels "github.com/Rahmannugar/consumel-server/internal/core/consumption/models"
	consumptiondb "github.com/Rahmannugar/consumel-server/internal/core/consumption/repositories/generated"
	"github.com/Rahmannugar/consumel-server/internal/infra/events"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AllowanceRepository struct{ pool *pgxpool.Pool }

func NewAllowanceRepository(pool *pgxpool.Pool) *AllowanceRepository {
	return &AllowanceRepository{pool: pool}
}

func (r *AllowanceRepository) Save(ctx context.Context, env uuid.UUID, customer, meter string, source consumptionmodels.BalanceMutationSource, input consumptionmodels.AllowanceRequest) (consumptionmodels.Allowance, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return consumptionmodels.Allowance{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := consumptiondb.New(tx)
	subject, err := balanceSubject(ctx, q, env, customer, meter)
	if err != nil {
		return consumptionmodels.Allowance{}, err
	}
	balanceID, err := ids.New()
	if err != nil {
		return consumptionmodels.Allowance{}, err
	}
	balance, err := q.EnsureBalance(ctx, consumptiondb.EnsureBalanceParams{ID: balanceID, ProjectEnvironmentID: env, CustomerID: subject.CustomerID, MeterID: subject.MeterID})
	if err != nil {
		return consumptionmodels.Allowance{}, err
	}
	if _, err = q.BalanceForUpdate(ctx, balance.ID); err != nil {
		return consumptionmodels.Allowance{}, err
	}
	// Every allowance writer takes the same balance lock as Consume. Reconcile
	// the old period before scheduling a revision, retaining its original result.
	if err = reconcileAllowance(ctx, tx, q, balance.ID); err != nil {
		return consumptionmodels.Allowance{}, err
	}
	policy, err := q.AllowanceByBalance(ctx, balance.ID)
	var effective time.Time
	scheduleAnchor := input.StartsAt
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if input.Version != 0 {
			return consumptionmodels.Allowance{}, consumptionmodels.ErrAllowanceConflict
		}
		id, idErr := ids.New()
		if idErr != nil {
			return consumptionmodels.Allowance{}, idErr
		}
		policy, err = q.CreateAllowance(ctx, consumptiondb.CreateAllowanceParams{ID: id, BalanceID: balance.ID, AnchorAt: timestamp(&input.StartsAt)})
		effective = input.StartsAt
	case err != nil:
		return consumptionmodels.Allowance{}, err
	default:
		if policy.Version != input.Version || !policy.AnchorAt.Time.Equal(input.StartsAt) {
			return consumptionmodels.Allowance{}, consumptionmodels.ErrAllowanceConflict
		}
		// Preserve an unchanged calendar anchor, including a pending edit.
		// A different interval starts at the next renewal, never mid-period.
		latest, loadErr := q.AllowanceRevisionByVersion(ctx, consumptiondb.AllowanceRevisionByVersionParams{AllowanceID: policy.ID, Version: policy.Version})
		if loadErr != nil {
			return consumptionmodels.Allowance{}, loadErr
		}
		now, clockErr := q.DatabaseTime(ctx)
		if clockErr != nil {
			return consumptionmodels.Allowance{}, clockErr
		}
		current, currentErr := q.EffectiveAllowanceRevision(ctx, consumptiondb.EffectiveAllowanceRevisionParams{AllowanceID: policy.ID, EffectiveAt: now})
		switch {
		case currentErr == nil && current.IntervalUnit == input.IntervalUnit && current.IntervalCount == input.IntervalCount:
			scheduleAnchor = current.ScheduleAnchorAt.Time
		case currentErr != nil && !errors.Is(currentErr, pgx.ErrNoRows):
			return consumptionmodels.Allowance{}, currentErr
		case latest.IntervalUnit == input.IntervalUnit && latest.IntervalCount == input.IntervalCount:
			scheduleAnchor = latest.ScheduleAnchorAt.Time
		default:
			scheduleAnchor = policy.NextPeriodAt.Time
		}
		if _, _, _, scheduleErr := (consumptionmodels.AllowanceSchedule{Unit: input.IntervalUnit, Count: input.IntervalCount, Anchor: scheduleAnchor}).Period(policy.NextPeriodAt.Time); scheduleErr != nil {
			return consumptionmodels.Allowance{}, scheduleErr
		}
		policy, err = q.IncrementAllowanceVersion(ctx, consumptiondb.IncrementAllowanceVersionParams{ID: policy.ID, Version: input.Version})
		effective = policy.NextPeriodAt.Time
	}
	if err != nil {
		return consumptionmodels.Allowance{}, err
	}
	if err = q.InsertAllowanceRevision(ctx, consumptiondb.InsertAllowanceRevisionParams{AllowanceID: policy.ID, Version: policy.Version, EffectiveAt: timestamp(&effective), Quantity: input.Quantity, RolloverCap: input.RolloverCap, Enabled: input.Enabled, SourceType: source.Type, SourceID: source.ActorID, IntervalUnit: input.IntervalUnit, IntervalCount: input.IntervalCount, ScheduleAnchorAt: timestamp(&scheduleAnchor)}); err != nil {
		return consumptionmodels.Allowance{}, err
	}
	if err = reconcileAllowance(ctx, tx, q, balance.ID); err != nil {
		return consumptionmodels.Allowance{}, err
	}
	result, err := allowanceView(ctx, q, balance.ID)
	if err != nil {
		return consumptionmodels.Allowance{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return consumptionmodels.Allowance{}, err
	}
	return result, nil
}
func (r *AllowanceRepository) Get(ctx context.Context, env uuid.UUID, customer, meter string) (consumptionmodels.Allowance, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return consumptionmodels.Allowance{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := consumptiondb.New(tx)
	balance, err := q.BalanceByPublicKeys(ctx, consumptiondb.BalanceByPublicKeysParams{ProjectEnvironmentID: env, CustomerID: customer, MeterKey: meter})
	if errors.Is(err, pgx.ErrNoRows) {
		return consumptionmodels.Allowance{}, consumptionmodels.ErrAllowanceNotFound
	}
	if err != nil {
		return consumptionmodels.Allowance{}, err
	}
	if _, err = q.BalanceForUpdate(ctx, balance.ID); err != nil {
		return consumptionmodels.Allowance{}, err
	}
	if err = reconcileAllowance(ctx, tx, q, balance.ID); err != nil {
		return consumptionmodels.Allowance{}, err
	}
	result, err := allowanceView(ctx, q, balance.ID)
	if err != nil {
		return consumptionmodels.Allowance{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return consumptionmodels.Allowance{}, err
	}
	return result, nil
}
func allowanceView(ctx context.Context, q *consumptiondb.Queries, balanceID uuid.UUID) (consumptionmodels.Allowance, error) {
	p, err := q.AllowanceByBalance(ctx, balanceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return consumptionmodels.Allowance{}, consumptionmodels.ErrAllowanceNotFound
	}
	if err != nil {
		return consumptionmodels.Allowance{}, err
	}
	at, err := q.DatabaseTime(ctx)
	if err != nil {
		return consumptionmodels.Allowance{}, err
	}
	result := consumptionmodels.Allowance{ID: p.ID, StartsAt: p.AnchorAt.Time, NextPeriodAt: p.NextPeriodAt.Time, Version: p.Version}
	current, err := q.EffectiveAllowanceRevision(ctx, consumptiondb.EffectiveAllowanceRevisionParams{AllowanceID: p.ID, EffectiveAt: at})
	if err == nil {
		result.Current = allowanceRevision(current)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return result, err
	}
	pending, err := q.PendingAllowanceRevision(ctx, consumptiondb.PendingAllowanceRevisionParams{AllowanceID: p.ID, EffectiveAt: at})
	if err == nil {
		result.Pending = allowanceRevision(pending)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return result, err
	}
	return result, nil
}
func allowanceRevision(r consumptiondb.AllowanceRevision) *consumptionmodels.AllowanceRevision {
	return &consumptionmodels.AllowanceRevision{Version: r.Version, EffectiveAt: r.EffectiveAt.Time, Quantity: r.Quantity, RolloverCap: r.RolloverCap, Enabled: r.Enabled, IntervalUnit: r.IntervalUnit, IntervalCount: r.IntervalCount, ScheduleAnchorAt: r.ScheduleAnchorAt.Time}
}

// The caller holds the balance row. Period identity, grant creation, projection,
// schedule advancement and outbox publication state commit together. No network
// call or worker availability is needed to make the next allowance spendable.
func reconcileAllowance(ctx context.Context, tx pgx.Tx, q *consumptiondb.Queries, balanceID uuid.UUID) error {
	if _, err := q.CaptureEntitlementTime(ctx); err != nil {
		return err
	}
	p, err := q.AllowanceByBalance(ctx, balanceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	at, err := q.DatabaseTime(ctx)
	if err != nil {
		return err
	}
	if p.NextPeriodAt.Time.After(at.Time) {
		return nil
	}
	revision, err := q.EffectiveAllowanceRevision(ctx, consumptiondb.EffectiveAllowanceRevisionParams{AllowanceID: p.ID, EffectiveAt: at})
	if err != nil {
		return err
	}
	schedule := consumptionmodels.AllowanceSchedule{Unit: revision.IntervalUnit, Count: revision.IntervalCount, Anchor: revision.ScheduleAnchorAt.Time}
	start, end, index, err := schedule.Period(at.Time)
	if err != nil {
		return err
	}
	_, _, dueIndex, err := schedule.Period(p.NextPeriodAt.Time)
	if err != nil {
		return err
	}
	last, err := q.LastAllowancePeriod(ctx, p.ID)
	var carry int64
	skipped := index
	if err == nil {
		skipped = index - dueIndex
		if skipped < 0 {
			return consumptionmodels.ErrAllowanceInvalid
		}
		if revision.Enabled {
			carry = consumptionmodels.AdvanceCarry(last.RemainingQuantity, revision.Quantity, revision.RolloverCap, skipped)
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	quantity := int64(0)
	var grantID pgtype.UUID
	if revision.Enabled {
		quantity = revision.Quantity + carry
		// Manual grants remain intact. Reject overflow atomically, preserving due
		// state for recovery rather than dropping an allowance or wrapping bigint.
		available, summaryErr := q.ActiveEntitlementSummary(ctx, balanceID)
		if summaryErr != nil {
			return summaryErr
		}
		if available.Quantity > math.MaxInt64-quantity {
			return consumptionmodels.ErrBalanceOverflow
		}
		id, idErr := ids.New()
		if idErr != nil {
			return idErr
		}
		if _, err = q.InsertAllowanceGrant(ctx, consumptiondb.InsertAllowanceGrantParams{ID: id, BalanceID: balanceID, GrantedQuantity: quantity, ExpiresAt: timestamp(&end)}); err != nil {
			return err
		}
		grantID = pgtype.UUID{Bytes: id, Valid: true}
	}
	if err = q.InsertAllowancePeriod(ctx, consumptiondb.InsertAllowancePeriodParams{AllowanceID: p.ID, PeriodStart: timestamp(&start), PeriodEnd: timestamp(&end), Revision: revision.Version, Quantity: quantity, CarriedQuantity: carry, SkippedPeriods: skipped, GrantID: grantID}); err != nil {
		return err
	}
	if err = q.AdvanceAllowance(ctx, consumptiondb.AdvanceAllowanceParams{ID: p.ID, NextPeriodAt: timestamp(&end)}); err != nil {
		return err
	}
	if _, err = q.RefreshBalanceProjection(ctx, balanceID); err != nil {
		return err
	}
	id, err := ids.New()
	if err != nil {
		return err
	}
	payload, err := json.Marshal(struct {
		AllowanceID     uuid.UUID `json:"allowanceId"`
		BalanceID       uuid.UUID `json:"balanceId"`
		PeriodStart     time.Time `json:"periodStart"`
		PeriodEnd       time.Time `json:"periodEnd"`
		Quantity        int64     `json:"quantity"`
		CarriedQuantity int64     `json:"carriedQuantity"`
		SkippedPeriods  int64     `json:"skippedPeriods"`
	}{p.ID, balanceID, start, end, quantity, carry, skipped})
	if err != nil {
		return err
	}
	return events.Insert(ctx, tx, events.Event{ID: id, Type: "allowance.period.recorded.v1", AggregateType: "recurring_allowance", AggregateID: p.ID, Payload: payload, OccurredAt: at.Time})
}
func (r *AllowanceRepository) Renew(ctx context.Context, balanceID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := consumptiondb.New(tx)
	if _, err = q.ActiveAllowanceBalanceForUpdate(ctx, balanceID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	if err = reconcileAllowance(ctx, tx, q, balanceID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *AllowanceRepository) Due(ctx context.Context, limit int) ([]consumptionmodels.AllowanceWork, error) {
	rows, err := consumptiondb.New(r.pool).DueAllowanceBalances(ctx, int32(limit))
	if err != nil {
		return nil, err
	}
	result := make([]consumptionmodels.AllowanceWork, 0, len(rows))
	for _, row := range rows {
		result = append(result, consumptionmodels.AllowanceWork{BalanceID: row.BalanceID, Attempt: int(row.RetryAttempts) + 1})
	}
	return result, nil
}
func (r *AllowanceRepository) RetryLater(ctx context.Context, id uuid.UUID) error {
	return consumptiondb.New(r.pool).DelayAllowanceRetry(ctx, id)
}

func (r *BalanceRepository) reconcileCustomerAllowances(ctx context.Context, env uuid.UUID, customer string) error {
	ids, err := consumptiondb.New(r.pool).CustomerAllowanceBalances(ctx, consumptiondb.CustomerAllowanceBalancesParams{ProjectEnvironmentID: env, CustomerID: customer})
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err = NewAllowanceRepository(r.pool).Renew(ctx, id); err != nil {
			return fmt.Errorf("renew customer allowance: %w", err)
		}
	}
	return nil
}

func (r *AllowanceRepository) Periods(ctx context.Context, env uuid.UUID, customer, meter string, cursor *consumptionmodels.OperationListCursor, limit int) (consumptionmodels.AllowancePeriods, error) {
	policy, err := r.Get(ctx, env, customer, meter)
	if err != nil {
		return consumptionmodels.AllowancePeriods{}, err
	}
	params := consumptiondb.ListAllowancePeriodsParams{AllowanceID: policy.ID, PageSize: int32(limit + 1)}
	if cursor != nil {
		if cursor.ID != policy.ID {
			return consumptionmodels.AllowancePeriods{}, consumptionmodels.ErrAllowanceCursorInvalid
		}
		params.BeforeStart = timestamp(&cursor.CreatedAt)
	}
	rows, err := consumptiondb.New(r.pool).ListAllowancePeriods(ctx, params)
	if err != nil {
		return consumptionmodels.AllowancePeriods{}, err
	}
	result := consumptionmodels.AllowancePeriods{Periods: make([]consumptionmodels.AllowancePeriodRecord, 0, min(limit, len(rows)))}
	for i, row := range rows {
		if i == limit {
			last := result.Periods[len(result.Periods)-1]
			encoded := consumptionmodels.EncodeOperationCursor(consumptionmodels.OperationListCursor{CreatedAt: last.PeriodStart, ID: policy.ID})
			result.NextCursor = &encoded
			break
		}
		result.Periods = append(result.Periods, consumptionmodels.AllowancePeriodRecord{PeriodStart: row.PeriodStart.Time, PeriodEnd: row.PeriodEnd.Time, Revision: row.Revision, Quantity: row.Quantity, CarriedQuantity: row.CarriedQuantity, SkippedPeriods: row.SkippedPeriods, SourceType: row.SourceType, SourceID: row.SourceID})
	}
	return result, nil
}

func (r *BalanceRepository) reconcileSubjectAllowance(ctx context.Context, env uuid.UUID, customer, meter string) error {
	id, err := consumptiondb.New(r.pool).DueSubjectAllowanceBalance(ctx, consumptiondb.DueSubjectAllowanceBalanceParams{ProjectEnvironmentID: env, CustomerID: customer, MeterKey: meter})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return NewAllowanceRepository(r.pool).Renew(ctx, id)
}
