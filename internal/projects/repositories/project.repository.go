package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Rahmannugar/consumel-server/internal/projects/models"
	projectdb "github.com/Rahmannugar/consumel-server/internal/projects/repositories/generated"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProjectRepository struct {
	pool    *pgxpool.Pool
	queries *projectdb.Queries
}

func nullableTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func NewProjectRepository(pool *pgxpool.Pool) *ProjectRepository {
	return &ProjectRepository{
		pool:    pool,
		queries: projectdb.New(pool),
	}
}

func (repository *ProjectRepository) ListAccessibleProjects(
	ctx context.Context,
	userID uuid.UUID,
) ([]models.Project, error) {
	rows, err := repository.queries.ListAccessibleProjects(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list accessible projects: %w", err)
	}
	projects := make([]models.Project, 0)
	for _, row := range rows {
		if len(projects) == 0 || projects[len(projects)-1].ID != row.ID {
			projects = append(projects, models.Project{
				ID: row.ID, OrganizationID: row.OrganizationID,
				OrganizationName: row.OrganizationName, Name: row.Name, Slug: row.Slug,
				CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
			})
		}
		current := &projects[len(projects)-1]
		current.Environments = append(current.Environments, models.ProjectEnvironment{
			ID: row.EnvironmentID, ProjectID: row.ID,
			Name:        models.ProjectEnvironmentName(row.Environment),
			ActivatedAt: nullableTime(row.EnvironmentActivatedAt),
			CreatedAt:   row.EnvironmentCreatedAt.Time,
		})
	}
	return projects, nil
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
		Slug:           project.Slug,
	})
	if projectNameExists(err) {
		return models.Project{}, models.ErrProjectNameExists
	}
	if errors.Is(err, pgx.ErrNoRows) {
		nameExists, lookupErr := queries.ProjectNameExists(ctx, projectdb.ProjectNameExistsParams{
			OrganizationID: project.OrganizationID,
			ProjectName:    project.Name,
		})
		if lookupErr != nil {
			return models.Project{}, fmt.Errorf("check project name: %w", lookupErr)
		}
		if nameExists {
			return models.Project{}, models.ErrProjectNameExists
		}
		project.Slug = models.ProjectSlugWithIDSuffix(project.Slug, project.ID)
		createdProject, err = queries.CreateProject(ctx, projectdb.CreateProjectParams{
			ID: project.ID, OrganizationID: project.OrganizationID,
			Name: project.Name, Slug: project.Slug,
		})
		if projectNameExists(err) {
			return models.Project{}, models.ErrProjectNameExists
		}
	}
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
		Slug:           createdProject.Slug,
		CreatedAt:      createdProject.CreatedAt.Time,
		UpdatedAt:      createdProject.UpdatedAt.Time,
		Environments:   createdEnvironments,
	}, nil
}

func projectNameExists(err error) bool {
	var databaseError *pgconn.PgError
	return errors.As(err, &databaseError) &&
		databaseError.ConstraintName == "projects_organization_name_unique_idx"
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
