package jobs

import (
	"context"
	"log/slog"
	"time"

	consumptionmodels "github.com/Rahmannugar/consumel-server/internal/core/consumption/models"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

// These short database transactions use two slots, leaving capacity for the
// worker's independent email and event pools. Each poll selects only work that
// can start immediately; the balance lock and period key make overlap safe.
const allowanceConcurrency = 2
const allowancePollInterval = 15 * time.Second

type AllowanceRepository interface {
	Due(context.Context, int) ([]consumptionmodels.AllowanceWork, error)
	Renew(context.Context, uuid.UUID) error
	RetryLater(context.Context, uuid.UUID) error
}
type JobObserver interface {
	Run(context.Context, string, int, func(context.Context) error) error
}
type AllowanceWorker struct {
	repository AllowanceRepository
	observer   JobObserver
	logger     *slog.Logger
}

func NewAllowanceWorker(repository AllowanceRepository, observer JobObserver, logger *slog.Logger) *AllowanceWorker {
	return &AllowanceWorker{repository: repository, observer: observer, logger: logger}
}
func (w *AllowanceWorker) Run(ctx context.Context) error {
	for ctx.Err() == nil {
		batchCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		work, err := w.repository.Due(batchCtx, allowanceConcurrency)
		if err != nil && ctx.Err() == nil {
			w.logger.ErrorContext(ctx, "Could not find due allowances", "event", "allowance.renew.scan.failed", "error_category", "database_unavailable")
		}
		group, groupCtx := errgroup.WithContext(batchCtx)
		for _, item := range work {
			group.Go(func() error {
				err := w.observer.Run(groupCtx, item.BalanceID.String(), item.Attempt, func(jobCtx context.Context) error { return w.repository.Renew(jobCtx, item.BalanceID) })
				if err != nil && groupCtx.Err() == nil {
					if retryErr := w.repository.RetryLater(groupCtx, item.BalanceID); retryErr != nil {
						w.logger.ErrorContext(groupCtx, "Could not schedule allowance retry", "event", "allowance.renew.retry.failed", "job_id", item.BalanceID.String(), "attempt", item.Attempt, "error_category", "database_unavailable")
					} else {
						w.logger.WarnContext(groupCtx, "Allowance renewal will retry", "event", "allowance.renew.retry", "job_id", item.BalanceID.String(), "attempt", item.Attempt, "retry_after_seconds", 60)
					}
				}
				return nil
			})
		}
		_ = group.Wait()
		cancel()
		if err == nil && len(work) == allowanceConcurrency {
			continue
		}
		timer := time.NewTimer(allowancePollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return ctx.Err()
}
