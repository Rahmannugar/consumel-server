package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Rahmannugar/consumel-server/internal/common/ids"
	"github.com/Rahmannugar/consumel-server/internal/projects/models"
	"github.com/google/uuid"
)

var (
	ErrOrganizationIDRequired = errors.New("organization ID is required")
	ErrUserIDRequired         = errors.New("user ID is required")
)

type ProjectRepository interface {
	CreateProjectWithEnvironments(
		context.Context,
		models.Project,
		[]models.ProjectEnvironment,
	) (models.Project, error)
	ListAccessibleProjects(context.Context, uuid.UUID) ([]models.Project, error)
	ActiveAPIKey(
		context.Context,
		uuid.UUID,
		uuid.UUID,
		models.ProjectEnvironmentName,
	) (*models.APIKey, error)
	CreateAPIKey(
		context.Context,
		uuid.UUID,
		uuid.UUID,
		models.ProjectEnvironmentName,
		models.APIKey,
		[]byte,
	) (models.APIKey, error)
	ReplaceAPIKey(
		context.Context,
		uuid.UUID,
		uuid.UUID,
		models.ProjectEnvironmentName,
		models.APIKey,
		[]byte,
	) (models.APIKey, error)
	RevokeAPIKey(
		context.Context,
		uuid.UUID,
		uuid.UUID,
		models.ProjectEnvironmentName,
	) error
	ActivateEnvironment(
		context.Context,
		uuid.UUID,
		uuid.UUID,
		models.ProjectEnvironmentName,
	) (models.ProjectEnvironment, error)
	AccessibleEnvironment(
		context.Context,
		uuid.UUID,
		uuid.UUID,
		models.ProjectEnvironmentName,
	) (models.ProjectEnvironment, error)
}

func (service *ProjectManagementService) ListProjects(
	ctx context.Context,
	userID uuid.UUID,
) ([]models.Project, error) {
	if userID == uuid.Nil {
		return nil, ErrUserIDRequired
	}
	return service.repository.ListAccessibleProjects(ctx, userID)
}

type ProjectManagementService struct {
	repository ProjectRepository
}

func NewProjectManagementService(repository ProjectRepository) *ProjectManagementService {
	return &ProjectManagementService{repository: repository}
}

func (service *ProjectManagementService) AccessibleEnvironment(
	ctx context.Context,
	userID uuid.UUID,
	projectID uuid.UUID,
	environment models.ProjectEnvironmentName,
) (models.ProjectEnvironment, error) {
	if err := validateAPIKeyContext(userID, projectID, environment); err != nil {
		return models.ProjectEnvironment{}, err
	}
	return service.repository.AccessibleEnvironment(ctx, userID, projectID, environment)
}

func (service *ProjectManagementService) CreateProject(
	ctx context.Context,
	organizationID uuid.UUID,
	name string,
) (models.Project, error) {
	if organizationID == uuid.Nil {
		return models.Project{}, ErrOrganizationIDRequired
	}

	request, err := (models.CreateProjectRequest{Name: name}).Validate()
	if err != nil {
		return models.Project{}, err
	}
	name = request.Name

	projectID, err := ids.New()
	if err != nil {
		return models.Project{}, fmt.Errorf("generate project ID: %w", err)
	}
	sandboxID, err := ids.New()
	if err != nil {
		return models.Project{}, fmt.Errorf("generate sandbox environment ID: %w", err)
	}
	liveID, err := ids.New()
	if err != nil {
		return models.Project{}, fmt.Errorf("generate live environment ID: %w", err)
	}

	activatedAt := time.Now().UTC()
	project := models.Project{
		ID: projectID, OrganizationID: organizationID, Name: name,
		Slug: models.NewProjectSlug(name, projectID),
	}
	environments := []models.ProjectEnvironment{
		{ID: sandboxID, ProjectID: projectID, Name: models.ProjectEnvironmentSandbox, ActivatedAt: &activatedAt},
		{ID: liveID, ProjectID: projectID, Name: models.ProjectEnvironmentLive},
	}

	return service.repository.CreateProjectWithEnvironments(ctx, project, environments)
}
