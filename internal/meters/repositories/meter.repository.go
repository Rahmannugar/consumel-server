package repositories

import (
	"context"
	"errors"
	"fmt"

	metermodels "github.com/Rahmannugar/consumel-server/internal/meters/models"
	meterdb "github.com/Rahmannugar/consumel-server/internal/meters/repositories/generated"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MeterRepository struct {
	queries *meterdb.Queries
}

func NewMeterRepository(pool *pgxpool.Pool) *MeterRepository {
	return &MeterRepository{queries: meterdb.New(pool)}
}

func (repository *MeterRepository) Create(
	ctx context.Context,
	meter metermodels.Meter,
) (metermodels.Meter, error) {
	row, err := repository.queries.CreateMeter(ctx, meterdb.CreateMeterParams{
		ProjectEnvironmentID: meter.ProjectEnvironmentID,
		MeterID:              meter.ID,
		MeterKey:             meter.MeterKey,
		Name:                 meter.Name,
		Description:          meter.Description,
		MeterType:            string(meter.Type),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return metermodels.Meter{}, metermodels.ErrMeterDefinitionConflict
	}
	if meterExists(err) {
		return metermodels.Meter{}, metermodels.ErrMeterExists
	}
	if err != nil {
		return metermodels.Meter{}, fmt.Errorf("create meter: %w", err)
	}
	return metermodels.Meter{
		ID: row.ID, ProjectID: row.ProjectID, ProjectEnvironmentID: row.ProjectEnvironmentID,
		MeterKey: row.MeterKey, Name: row.Name, Description: row.Description,
		Type: metermodels.MeterType(row.MeterType), CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}, nil
}

func (repository *MeterRepository) Get(
	ctx context.Context,
	projectEnvironmentID uuid.UUID,
	meterKey string,
) (metermodels.Meter, error) {
	row, err := repository.queries.MeterByPublicKey(ctx, meterdb.MeterByPublicKeyParams{
		ProjectEnvironmentID: projectEnvironmentID, MeterKey: meterKey,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return metermodels.Meter{}, metermodels.ErrMeterNotFound
	}
	if err != nil {
		return metermodels.Meter{}, fmt.Errorf("get meter: %w", err)
	}
	return metermodels.Meter{
		ID: row.ID, ProjectID: row.ProjectID, ProjectEnvironmentID: row.ProjectEnvironmentID,
		MeterKey: row.MeterKey, Name: row.Name, Description: row.Description,
		Type: metermodels.MeterType(row.MeterType), CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}, nil
}

func (repository *MeterRepository) List(
	ctx context.Context,
	projectEnvironmentID uuid.UUID,
	cursor *metermodels.ListCursor,
	limit int,
	search string,
) ([]metermodels.Meter, *metermodels.ListCursor, error) {
	params := meterdb.ListMetersParams{
		ProjectEnvironmentID: projectEnvironmentID, PageSize: int32(limit + 1),
	}
	if cursor != nil {
		params.CursorCreatedAt = pgtype.Timestamptz{Time: cursor.CreatedAt, Valid: true}
		params.CursorID = pgtype.UUID{Bytes: cursor.ID, Valid: true}
	}
	if search != "" {
		return repository.search(ctx, params, search, limit)
	}
	rows, err := repository.queries.ListMeters(ctx, params)
	if err != nil {
		return nil, nil, fmt.Errorf("list meters: %w", err)
	}
	meters := make([]metermodels.Meter, 0, min(len(rows), limit))
	for index, row := range rows {
		if index == limit {
			last := meters[len(meters)-1]
			return meters, &metermodels.ListCursor{CreatedAt: last.CreatedAt, ID: last.ID}, nil
		}
		meters = append(meters, metermodels.Meter{
			ID: row.ID, ProjectID: row.ProjectID, ProjectEnvironmentID: row.ProjectEnvironmentID,
			MeterKey: row.MeterKey, Name: row.Name, Description: row.Description,
			Type: metermodels.MeterType(row.MeterType), CreatedAt: row.CreatedAt.Time,
			UpdatedAt: row.UpdatedAt.Time,
		})
	}
	return meters, nil, nil
}

func (repository *MeterRepository) search(
	ctx context.Context,
	params meterdb.ListMetersParams,
	search string,
	limit int,
) ([]metermodels.Meter, *metermodels.ListCursor, error) {
	rows, err := repository.queries.SearchMeters(ctx, meterdb.SearchMetersParams{
		ProjectEnvironmentID: params.ProjectEnvironmentID,
		SearchQuery:          search, CursorCreatedAt: params.CursorCreatedAt,
		CursorID: params.CursorID, PageSize: params.PageSize,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("search meters: %w", err)
	}
	meters := make([]metermodels.Meter, 0, min(len(rows), limit))
	for index, row := range rows {
		if index == limit {
			last := meters[len(meters)-1]
			return meters, &metermodels.ListCursor{CreatedAt: last.CreatedAt, ID: last.ID}, nil
		}
		meters = append(meters, metermodels.Meter{
			ID: row.ID, ProjectID: row.ProjectID, ProjectEnvironmentID: row.ProjectEnvironmentID,
			MeterKey: row.MeterKey, Name: row.Name, Description: row.Description,
			Type: metermodels.MeterType(row.MeterType), CreatedAt: row.CreatedAt.Time,
			UpdatedAt: row.UpdatedAt.Time,
		})
	}
	return meters, nil, nil
}

func meterExists(err error) bool {
	var databaseError *pgconn.PgError
	return errors.As(err, &databaseError) &&
		databaseError.ConstraintName == "project_environment_meters_pkey"
}
