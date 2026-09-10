package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Rahmannugar/consumel-server/internal/common/ids"
	"github.com/Rahmannugar/consumel-server/internal/projects/models"
	"github.com/google/uuid"
)

var (
	ErrOrganizationIDRequired = errors.New("organization ID is required")
	ErrProjectNameRequired    = errors.New("project name is required")
)

type ProjectRepository interface {
	CreateProjectWithEnvironments(
		context.Context,
		models.Project,
		[]models.ProjectEnvironment,
	) (models.Project, error)
}

type ProjectManagementService struct {
	repository ProjectRepository
}

func NewProjectManagementService(repository ProjectRepository) *ProjectManagementService {
	return &ProjectManagementService{repository: repository}
}

func (service *ProjectManagementService) CreateProject(
	ctx context.Context,
	organizationID uuid.UUID,
	name string,
) (models.Project, error) {
	if organizationID == uuid.Nil {
		return models.Project{}, ErrOrganizationIDRequired
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return models.Project{}, ErrProjectNameRequired
	}

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
	project := models.Project{ID: projectID, OrganizationID: organizationID, Name: name}
	environments := []models.ProjectEnvironment{
		{ID: sandboxID, ProjectID: projectID, Name: models.ProjectEnvironmentSandbox, ActivatedAt: &activatedAt},
		{ID: liveID, ProjectID: projectID, Name: models.ProjectEnvironmentLive},
	}

	return service.repository.CreateProjectWithEnvironments(ctx, project, environments)
}
