package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/Rahmannugar/consumel-server/internal/common/ids"
	metermodels "github.com/Rahmannugar/consumel-server/internal/meters/models"
	"github.com/google/uuid"
)

const (
	DefaultPageSize = 25
	MaximumPageSize = 100
)

type MeterRepository interface {
	Create(context.Context, metermodels.Meter) (metermodels.Meter, error)
	Get(context.Context, uuid.UUID, string) (metermodels.Meter, error)
	List(context.Context, uuid.UUID, *metermodels.ListCursor, int) ([]metermodels.Meter, *metermodels.ListCursor, error)
}

type MeterService struct {
	repository MeterRepository
}

func NewMeterService(repository MeterRepository) *MeterService {
	return &MeterService{repository: repository}
}

func (service *MeterService) Create(
	ctx context.Context,
	projectEnvironmentID uuid.UUID,
	request metermodels.CreateMeterRequest,
) (metermodels.Meter, error) {
	request, err := request.Validate()
	if err != nil {
		return metermodels.Meter{}, err
	}
	id, err := ids.New()
	if err != nil {
		return metermodels.Meter{}, fmt.Errorf("generate meter ID: %w", err)
	}
	return service.repository.Create(ctx, metermodels.Meter{
		ID: id, ProjectEnvironmentID: projectEnvironmentID, MeterKey: request.MeterKey,
		Name: request.Name, Description: request.Description, Type: request.Type,
	})
}

func (service *MeterService) Get(
	ctx context.Context,
	projectEnvironmentID uuid.UUID,
	meterKey string,
) (metermodels.Meter, error) {
	meterKey, err := validateMeterKey(meterKey)
	if err != nil {
		return metermodels.Meter{}, err
	}
	return service.repository.Get(ctx, projectEnvironmentID, meterKey)
}

func (service *MeterService) List(
	ctx context.Context,
	projectEnvironmentID uuid.UUID,
	cursor *metermodels.ListCursor,
	limit int,
) ([]metermodels.Meter, *metermodels.ListCursor, error) {
	if limit <= 0 {
		limit = DefaultPageSize
	}
	if limit > MaximumPageSize {
		limit = MaximumPageSize
	}
	return service.repository.List(ctx, projectEnvironmentID, cursor, limit)
}

func validateMeterKey(value string) (string, error) {
	request, err := (metermodels.CreateMeterRequest{
		MeterKey: strings.TrimSpace(value), Name: "validation", Type: metermodels.MeterTypePrepaid,
	}).Validate()
	if err != nil {
		return "", err
	}
	return request.MeterKey, nil
}
