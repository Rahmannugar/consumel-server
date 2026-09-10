package repositories

import (
	"context"
	"fmt"

	"github.com/Rahmannugar/consumel-server/internal/projects/models"
	projectdb "github.com/Rahmannugar/consumel-server/internal/projects/repositories/generated"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProjectRepository struct {
	pool    *pgxpool.Pool
	queries *projectdb.Queries
}

func NewProjectRepository(pool *pgxpool.Pool) *ProjectRepository {
	return &ProjectRepository{
		pool:    pool,
		queries: projectdb.New(pool),
	}
}

func (repository *ProjectRepository) CreateProjectWithEnvironments(
	ctx context.Context,
	project models.Project,
	environments []models.ProjectEnvironment,
) (models.Project, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return models.Project{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	queries := repository.queries.WithTx(tx)
	createdProject, err := queries.CreateProject(ctx, projectdb.CreateProjectParams{
		ID:             project.ID,
		OrganizationID: project.OrganizationID,
		Name:           project.Name,
	})
	if err != nil {
		return models.Project{}, fmt.Errorf("create project: %w", err)
	}

	createdEnvironments := make([]models.ProjectEnvironment, 0, len(environments))
	for _, environment := range environments {
		createdEnvironment, err := createProjectEnvironment(ctx, queries, environment)
		if err != nil {
			return models.Project{}, err
		}
		createdEnvironments = append(createdEnvironments, createdEnvironment)
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Project{}, fmt.Errorf("commit transaction: %w", err)
	}

	return models.Project{
		ID:             createdProject.ID,
		OrganizationID: createdProject.OrganizationID,
		Name:           createdProject.Name,
		CreatedAt:      createdProject.CreatedAt.Time,
		UpdatedAt:      createdProject.UpdatedAt.Time,
		Environments:   createdEnvironments,
	}, nil
}

func mapProjectEnvironment(
	environment projectdb.ProjectEnvironment,
) models.ProjectEnvironment {
	return models.ProjectEnvironment{
		ID:          environment.ID,
		ProjectID:   environment.ProjectID,
		Name:        models.ProjectEnvironmentName(environment.Environment),
		ActivatedAt: projectEnvironmentActivation(environment),
		CreatedAt:   environment.CreatedAt.Time,
	}
}
