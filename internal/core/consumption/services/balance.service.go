package services

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Rahmannugar/consumel-server/internal/common/ids"
	consumptionmodels "github.com/Rahmannugar/consumel-server/internal/core/consumption/models"
	customermodels "github.com/Rahmannugar/consumel-server/internal/customers/models"
	metermodels "github.com/Rahmannugar/consumel-server/internal/meters/models"
	"github.com/google/uuid"
)

type BalanceRepository interface {
	Add(context.Context, uuid.UUID, consumptionmodels.AddBalanceRequest, uuid.UUID, uuid.UUID, uuid.UUID) (consumptionmodels.Balance, bool, error)
	Set(context.Context, uuid.UUID, string, string, int64, uuid.UUID, uuid.UUID) (consumptionmodels.Balance, error)
	Get(context.Context, uuid.UUID, string, string) (consumptionmodels.Balance, error)
	List(context.Context, uuid.UUID, string) ([]consumptionmodels.Balance, error)
}

type BalanceService struct {
	repository BalanceRepository
}

func NewBalanceService(repository BalanceRepository) *BalanceService {
	return &BalanceService{repository: repository}
}

func (service *BalanceService) Add(
	ctx context.Context,
	projectEnvironmentID uuid.UUID,
	idempotencyKey string,
	request consumptionmodels.AddBalanceRequest,
) (consumptionmodels.Balance, bool, error) {
	request, err := request.Validate()
	if err != nil {
		return consumptionmodels.Balance{}, false, err
	}
	key, err := uuid.Parse(strings.TrimSpace(idempotencyKey))
	if err != nil || key.Version() != 7 {
		return consumptionmodels.Balance{}, false, consumptionmodels.ErrIdempotencyKeyInvalid
	}
	operationID, balanceID, err := newOperationAndBalanceIDs()
	if err != nil {
		return consumptionmodels.Balance{}, false, err
	}
	return service.repository.Add(ctx, projectEnvironmentID, request, key, operationID, balanceID)
}

func (service *BalanceService) Set(
	ctx context.Context,
	projectEnvironmentID uuid.UUID,
	customerID, meterKey string,
	request consumptionmodels.SetBalanceRequest,
) (consumptionmodels.Balance, error) {
	customerID, meterKey, err := validateSubjectKeys(customerID, meterKey)
	if err != nil {
		return consumptionmodels.Balance{}, err
	}
	request, err = request.Validate()
	if err != nil {
		return consumptionmodels.Balance{}, err
	}
	operationID, balanceID, err := newOperationAndBalanceIDs()
	if err != nil {
		return consumptionmodels.Balance{}, err
	}
	return service.repository.Set(ctx, projectEnvironmentID, customerID, meterKey, request.Quantity, operationID, balanceID)
}

func (service *BalanceService) Get(
	ctx context.Context,
	projectEnvironmentID uuid.UUID,
	customerID, meterKey string,
) (consumptionmodels.Balance, error) {
	customerID, meterKey, err := validateSubjectKeys(customerID, meterKey)
	if err != nil {
		return consumptionmodels.Balance{}, err
	}
	return service.repository.Get(ctx, projectEnvironmentID, customerID, meterKey)
}

func (service *BalanceService) List(
	ctx context.Context,
	projectEnvironmentID uuid.UUID,
	customerID string,
) ([]consumptionmodels.Balance, error) {
	customerID = strings.TrimSpace(customerID)
	if customerID == "" || utf8.RuneCountInString(customerID) > customermodels.MaximumCustomerIDLength {
		return nil, consumptionmodels.ErrBalanceCustomerInvalid
	}
	return service.repository.List(ctx, projectEnvironmentID, customerID)
}

func validateSubjectKeys(customerID, meterKey string) (string, string, error) {
	customerID = strings.TrimSpace(customerID)
	meterKey = strings.TrimSpace(meterKey)
	if customerID == "" || utf8.RuneCountInString(customerID) > customermodels.MaximumCustomerIDLength {
		return "", "", consumptionmodels.ErrBalanceCustomerInvalid
	}
	if !metermodels.ValidMeterKey(meterKey) {
		return "", "", consumptionmodels.ErrBalanceMeterInvalid
	}
	return customerID, meterKey, nil
}

func newOperationAndBalanceIDs() (uuid.UUID, uuid.UUID, error) {
	operationID, err := ids.New()
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("generate balance operation ID: %w", err)
	}
	balanceID, err := ids.New()
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("generate balance ID: %w", err)
	}
	return operationID, balanceID, nil
}
