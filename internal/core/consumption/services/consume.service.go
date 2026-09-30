package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/Rahmannugar/consumel-server/internal/common/ids"
	consumptionmodels "github.com/Rahmannugar/consumel-server/internal/core/consumption/models"
	"github.com/google/uuid"
)

type ConsumeRepository interface {
	Consume(
		context.Context,
		uuid.UUID,
		consumptionmodels.ConsumeRequest,
		uuid.UUID,
		uuid.UUID,
		uuid.UUID,
		uuid.UUID,
		uuid.UUID,
	) (consumptionmodels.UsageEvent, bool, error)
}

type ConsumeService struct {
	repository ConsumeRepository
}

func NewConsumeService(repository ConsumeRepository) *ConsumeService {
	return &ConsumeService{repository: repository}
}

func (service *ConsumeService) Consume(
	ctx context.Context,
	projectEnvironmentID uuid.UUID,
	idempotencyKey string,
	request consumptionmodels.ConsumeRequest,
) (consumptionmodels.UsageEvent, bool, error) {
	request, err := request.Validate()
	if err != nil {
		return consumptionmodels.UsageEvent{}, false, err
	}
	key, err := uuid.Parse(strings.TrimSpace(idempotencyKey))
	if err != nil || key.Version() != 7 {
		return consumptionmodels.UsageEvent{}, false, consumptionmodels.ErrIdempotencyKeyInvalid
	}
	operationID, customerID, balanceID, outboxID, err := newConsumptionIDs()
	if err != nil {
		return consumptionmodels.UsageEvent{}, false, err
	}
	return service.repository.Consume(
		ctx, projectEnvironmentID, request, key, operationID, customerID, balanceID, outboxID,
	)
}

func newConsumptionIDs() (uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, error) {
	values := make([]uuid.UUID, 4)
	for index := range values {
		value, err := ids.New()
		if err != nil {
			return uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil,
				fmt.Errorf("generate consumption identifier: %w", err)
		}
		values[index] = value
	}
	return values[0], values[1], values[2], values[3], nil
}
