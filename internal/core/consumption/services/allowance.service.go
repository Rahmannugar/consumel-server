package services

import (
	"context"

	consumptionmodels "github.com/Rahmannugar/consumel-server/internal/core/consumption/models"
	"github.com/google/uuid"
)

type AllowanceRepository interface {
	Periods(context.Context, uuid.UUID, string, string, *consumptionmodels.OperationListCursor, int) (consumptionmodels.AllowancePeriods, error)
	Save(context.Context, uuid.UUID, string, string, consumptionmodels.BalanceMutationSource, consumptionmodels.AllowanceRequest) (consumptionmodels.Allowance, error)
	Get(context.Context, uuid.UUID, string, string) (consumptionmodels.Allowance, error)
}
type AllowanceService struct{ repository AllowanceRepository }

func NewAllowanceService(r AllowanceRepository) *AllowanceService {
	return &AllowanceService{repository: r}
}
func (s *AllowanceService) Save(ctx context.Context, env uuid.UUID, customer, meter string, source consumptionmodels.BalanceMutationSource, input consumptionmodels.AllowanceRequest) (consumptionmodels.Allowance, error) {
	customer, meter, err := validateSubjectKeys(customer, meter)
	if err != nil {
		return consumptionmodels.Allowance{}, err
	}
	input, err = input.Validate()
	if err != nil {
		return consumptionmodels.Allowance{}, err
	}
	if (source.Type != consumptionmodels.BalanceSourceAPIKey && source.Type != consumptionmodels.BalanceSourceDashboardUser) || source.ActorID == uuid.Nil {
		return consumptionmodels.Allowance{}, consumptionmodels.ErrAllowanceInvalid
	}
	return s.repository.Save(ctx, env, customer, meter, source, input)
}
func (s *AllowanceService) Get(ctx context.Context, env uuid.UUID, customer, meter string) (consumptionmodels.Allowance, error) {
	customer, meter, err := validateSubjectKeys(customer, meter)
	if err != nil {
		return consumptionmodels.Allowance{}, err
	}
	return s.repository.Get(ctx, env, customer, meter)
}

func (s *AllowanceService) Periods(ctx context.Context, env uuid.UUID, customer, meter string, cursor *consumptionmodels.OperationListCursor, limit int) (consumptionmodels.AllowancePeriods, error) {
	customer, meter, err := validateSubjectKeys(customer, meter)
	if err != nil {
		return consumptionmodels.AllowancePeriods{}, err
	}
	if limit <= 0 {
		limit = DefaultBalanceDetailPageSize
	}
	if limit > MaximumBalanceDetailPageSize {
		limit = MaximumBalanceDetailPageSize
	}
	return s.repository.Periods(ctx, env, customer, meter, cursor, limit)
}
