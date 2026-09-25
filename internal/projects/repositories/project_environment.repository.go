package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/Rahmannugar/consumel-server/internal/projects/models"
	projectdb "github.com/Rahmannugar/consumel-server/internal/projects/repositories/generated"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func (repository *ProjectRepository) ProjectEnvironments(
	ctx context.Context,
	projectID uuid.UUID,
) ([]models.ProjectEnvironment, error) {
	environments, err := repository.queries.ListProjectEnvironments(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list project environments: %w", err)
	}

	result := make([]models.ProjectEnvironment, 0, len(environments))
	for _, environment := range environments {
		result = append(result, mapProjectEnvironment(environment))
	}
	return result, nil
}

func createProjectEnvironment(
	ctx context.Context,
	queries *projectdb.Queries,
	environment models.ProjectEnvironment,
) (models.ProjectEnvironment, error) {
	created, err := queries.CreateProjectEnvironment(ctx, projectdb.CreateProjectEnvironmentParams{
		ID:          environment.ID,
		ProjectID:   environment.ProjectID,
		Environment: string(environment.Name),
		ActivatedAt: timestamp(environment.ActivatedAt),
	})
	if err != nil {
		return models.ProjectEnvironment{}, fmt.Errorf("create %s environment: %w", environment.Name, err)
	}

	return mapProjectEnvironment(created), nil
}

func timestamp(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *value, Valid: true}
}

func projectEnvironmentActivation(environment projectdb.ProjectEnvironment) *time.Time {
	if !environment.ActivatedAt.Valid {
		return nil
	}
	activatedAt := environment.ActivatedAt.Time
	return &activatedAt
}
