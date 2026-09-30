package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/Rahmannugar/consumel-server/internal/projects/models"
	projectdb "github.com/Rahmannugar/consumel-server/internal/projects/repositories/generated"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (repository *ProjectRepository) ActiveAPIKey(
	ctx context.Context,
	userID uuid.UUID,
	projectID uuid.UUID,
	environment models.ProjectEnvironmentName,
) (*models.APIKey, error) {
	projectEnvironment, err := repository.queries.AccessibleProjectEnvironment(
		ctx,
		projectdb.AccessibleProjectEnvironmentParams{
			ID: projectID, Environment: string(environment), UserID: userID,
		},
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, models.ErrProjectEnvironmentUnavailable
	}
	if err != nil {
		return nil, fmt.Errorf("authorize project environment: %w", err)
	}
	row, err := repository.queries.ActiveProjectAPIKey(ctx, projectEnvironment.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load active project API key: %w", err)
	}
	key := mapAPIKey(row, environment)
	return &key, nil
}

func (repository *ProjectRepository) CreateAPIKey(
	ctx context.Context,
	userID uuid.UUID,
	projectID uuid.UUID,
	environment models.ProjectEnvironmentName,
	key models.APIKey,
	hash []byte,
) (models.APIKey, error) {
	return repository.writeAPIKey(ctx, userID, projectID, environment, key, hash, false)
}

func (repository *ProjectRepository) ReplaceAPIKey(
	ctx context.Context,
	userID uuid.UUID,
	projectID uuid.UUID,
	environment models.ProjectEnvironmentName,
	key models.APIKey,
	hash []byte,
) (models.APIKey, error) {
	return repository.writeAPIKey(ctx, userID, projectID, environment, key, hash, true)
}

func (repository *ProjectRepository) writeAPIKey(
	ctx context.Context,
	userID uuid.UUID,
	projectID uuid.UUID,
	environment models.ProjectEnvironmentName,
	key models.APIKey,
	hash []byte,
	replace bool,
) (models.APIKey, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return models.APIKey{}, fmt.Errorf("begin API key transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	queries := repository.queries.WithTx(tx)
	projectEnvironment, err := lockProjectEnvironment(
		ctx, queries, userID, projectID, environment,
	)
	if err != nil {
		return models.APIKey{}, err
	}
	if !projectEnvironment.ActivatedAt.Valid {
		return models.APIKey{}, models.ErrProjectEnvironmentInactive
	}

	active, err := queries.ActiveProjectAPIKey(ctx, projectEnvironment.ID)
	if err == nil && !replace {
		return models.APIKey{}, models.ErrActiveAPIKeyExists
	}
	if errors.Is(err, pgx.ErrNoRows) && replace {
		return models.APIKey{}, models.ErrActiveAPIKeyNotFound
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return models.APIKey{}, fmt.Errorf("load active API key: %w", err)
	}
	if replace {
		_, err = queries.RevokeActiveProjectAPIKey(ctx, projectdb.RevokeActiveProjectAPIKeyParams{
			ProjectEnvironmentID: projectEnvironment.ID,
			RevokedByUserID:      pgtype.UUID{Bytes: userID, Valid: true},
		})
		if err != nil {
			return models.APIKey{}, fmt.Errorf("revoke replaced API key: %w", err)
		}
	}
	_ = active

	created, err := queries.CreateProjectAPIKey(ctx, projectdb.CreateProjectAPIKeyParams{
		ID: key.ID, ProjectEnvironmentID: projectEnvironment.ID,
		KeyHash: hash, KeyPrefix: key.Prefix, LastFour: key.LastFour,
		CreatedByUserID: userID,
	})
	if err != nil {
		return models.APIKey{}, fmt.Errorf("create project API key: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return models.APIKey{}, fmt.Errorf("commit API key transaction: %w", err)
	}
	return mapCreatedAPIKey(created, environment), nil
}

func (repository *ProjectRepository) RevokeAPIKey(
	ctx context.Context,
	userID uuid.UUID,
	projectID uuid.UUID,
	environment models.ProjectEnvironmentName,
) error {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin API key revocation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	queries := repository.queries.WithTx(tx)
	projectEnvironment, err := lockProjectEnvironment(
		ctx, queries, userID, projectID, environment,
	)
	if err != nil {
		return err
	}
	_, err = queries.RevokeActiveProjectAPIKey(ctx, projectdb.RevokeActiveProjectAPIKeyParams{
		ProjectEnvironmentID: projectEnvironment.ID,
		RevokedByUserID:      pgtype.UUID{Bytes: userID, Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return models.ErrActiveAPIKeyNotFound
	}
	if err != nil {
		return fmt.Errorf("revoke active API key: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit API key revocation: %w", err)
	}
	return nil
}

func lockProjectEnvironment(
	ctx context.Context,
	queries *projectdb.Queries,
	userID uuid.UUID,
	projectID uuid.UUID,
	environment models.ProjectEnvironmentName,
) (projectdb.ProjectEnvironment, error) {
	value, err := queries.LockAccessibleProjectEnvironment(
		ctx,
		projectdb.LockAccessibleProjectEnvironmentParams{
			ID: projectID, Environment: string(environment), UserID: userID,
		},
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return projectdb.ProjectEnvironment{}, models.ErrProjectEnvironmentUnavailable
	}
	if err != nil {
		return projectdb.ProjectEnvironment{}, fmt.Errorf("lock project environment: %w", err)
	}
	return value, nil
}

func mapAPIKey(
	row projectdb.ActiveProjectAPIKeyRow,
	environment models.ProjectEnvironmentName,
) models.APIKey {
	return models.APIKey{
		ID: row.ID, ProjectEnvironmentID: row.ProjectEnvironmentID,
		Environment: environment, Prefix: row.KeyPrefix, LastFour: row.LastFour,
		CreatedAt: row.CreatedAt.Time, LastUsedAt: nullableTime(row.LastUsedAt),
		RevokedAt: nullableTime(row.RevokedAt),
	}
}

func mapCreatedAPIKey(
	row projectdb.CreateProjectAPIKeyRow,
	environment models.ProjectEnvironmentName,
) models.APIKey {
	return models.APIKey{
		ID: row.ID, ProjectEnvironmentID: row.ProjectEnvironmentID,
		Environment: environment, Prefix: row.KeyPrefix, LastFour: row.LastFour,
		CreatedAt: row.CreatedAt.Time, LastUsedAt: nullableTime(row.LastUsedAt),
		RevokedAt: nullableTime(row.RevokedAt),
	}
}
