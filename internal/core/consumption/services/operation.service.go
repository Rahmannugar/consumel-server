package services

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	consumptionmodels "github.com/Rahmannugar/consumel-server/internal/core/consumption/models"
	customermodels "github.com/Rahmannugar/consumel-server/internal/customers/models"
	metermodels "github.com/Rahmannugar/consumel-server/internal/meters/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	DefaultOperationPageSize = 50
	MaximumOperationPageSize = 100
	maximumOperationRange    = 366 * 24 * time.Hour
	defaultOperationRange    = 7 * 24 * time.Hour
)

type OperationRepository interface {
	List(context.Context, uuid.UUID, *consumptionmodels.OperationListCursor, int, consumptionmodels.OperationListFilter) ([]consumptionmodels.Operation, *consumptionmodels.OperationListCursor, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (consumptionmodels.Operation, error)
}

type OperationStreamNotice struct {
	Cursor               string
	ProjectEnvironmentID uuid.UUID
	OperationID          uuid.UUID
}

type OperationStreamSource interface {
	Read(context.Context, string, time.Duration) ([]OperationStreamNotice, string, error)
}

type OperationService struct {
	repository OperationRepository
	stream     OperationStreamSource
	clock      func() time.Time
}

func NewOperationService(repository OperationRepository, stream OperationStreamSource) *OperationService {
	return &OperationService{repository: repository, stream: stream, clock: time.Now}
}

func (service *OperationService) Next(
	ctx context.Context,
	projectEnvironmentID uuid.UUID,
	after string,
	block time.Duration,
) (consumptionmodels.Operation, string, bool, error) {
	notices, latest, err := service.stream.Read(ctx, after, block)
	if err != nil {
		return consumptionmodels.Operation{}, after, false, err
	}
	for _, notice := range notices {
		if notice.ProjectEnvironmentID != projectEnvironmentID {
			continue
		}
		operation, err := service.repository.Get(ctx, projectEnvironmentID, notice.OperationID)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return consumptionmodels.Operation{}, notice.Cursor, false, err
		}
		return operation, notice.Cursor, true, nil
	}
	return consumptionmodels.Operation{}, latest, false, nil
}

func (service *OperationService) List(
	ctx context.Context,
	projectEnvironmentID uuid.UUID,
	cursor *consumptionmodels.OperationListCursor,
	limit int,
	filter consumptionmodels.OperationListFilter,
	fromValue string,
	toValue string,
) ([]consumptionmodels.Operation, *consumptionmodels.OperationListCursor, error) {
	if limit <= 0 {
		limit = DefaultOperationPageSize
	}
	if limit > MaximumOperationPageSize {
		limit = MaximumOperationPageSize
	}
	if filter.Status != "" && filter.Status != consumptionmodels.OperationStatusAccepted && filter.Status != consumptionmodels.OperationStatusDenied {
		return nil, nil, consumptionmodels.ErrOperationFilterInvalid
	}
	filter.CustomerID = strings.TrimSpace(filter.CustomerID)
	if utf8.RuneCountInString(filter.CustomerID) > customermodels.MaximumCustomerIDLength {
		return nil, nil, consumptionmodels.ErrOperationFilterInvalid
	}
	filter.MeterKey = strings.TrimSpace(filter.MeterKey)
	if filter.MeterKey != "" && !metermodels.ValidMeterKey(filter.MeterKey) {
		return nil, nil, consumptionmodels.ErrOperationFilterInvalid
	}
	from, to, err := service.operationRange(fromValue, toValue)
	if err != nil {
		return nil, nil, consumptionmodels.ErrOperationFilterInvalid
	}
	filter.From = from
	filter.To = to
	return service.repository.List(ctx, projectEnvironmentID, cursor, limit, filter)
}

func (service *OperationService) operationRange(fromValue, toValue string) (time.Time, time.Time, error) {
	now := service.clock().UTC()
	if fromValue == "" && toValue == "" {
		return now.Add(-defaultOperationRange), now, nil
	}
	if fromValue == "" || toValue == "" {
		return time.Time{}, time.Time{}, consumptionmodels.ErrOperationFilterInvalid
	}
	from, err := time.Parse(time.RFC3339, fromValue)
	if err != nil {
		return time.Time{}, time.Time{}, consumptionmodels.ErrOperationFilterInvalid
	}
	to, err := time.Parse(time.RFC3339, toValue)
	if err != nil || !from.Before(to) || to.Sub(from) > maximumOperationRange || to.After(now.Add(5*time.Minute)) {
		return time.Time{}, time.Time{}, consumptionmodels.ErrOperationFilterInvalid
	}
	return from.UTC(), to.UTC(), nil
}
