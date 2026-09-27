package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Rahmannugar/consumel-server/internal/common/ids"
	onboardingmodels "github.com/Rahmannugar/consumel-server/internal/onboarding/models"
	organizationmodels "github.com/Rahmannugar/consumel-server/internal/organizations/models"
	projectmodels "github.com/Rahmannugar/consumel-server/internal/projects/models"
	"github.com/google/uuid"
)

type Repository interface {
	SetupFirstProject(
		context.Context,
		string,
		organizationmodels.Organization,
		[]organizationmodels.OrganizationRole,
		organizationmodels.OrganizationMembership,
		projectmodels.Project,
		[]projectmodels.ProjectEnvironment,
		string,
	) (onboardingmodels.Setup, error)
}

type Service struct {
	repository    Repository
	clientBaseURL string
}

func New(repository Repository, clientBaseURL string) *Service {
	return &Service{
		repository:    repository,
		clientBaseURL: strings.TrimRight(clientBaseURL, "/"),
	}
}

func (service *Service) Setup(
	ctx context.Context,
	subjectID string,
	userID uuid.UUID,
	request onboardingmodels.SetupRequest,
) (onboardingmodels.Setup, error) {
	request, err := request.Validate()
	if err != nil {
		return onboardingmodels.Setup{}, err
	}
	if strings.TrimSpace(subjectID) == "" || userID == uuid.Nil {
		return onboardingmodels.Setup{}, fmt.Errorf("authenticated identity is incomplete")
	}

	organizationID, err := ids.New()
	if err != nil {
		return onboardingmodels.Setup{}, fmt.Errorf("generate organization ID: %w", err)
	}
	adminRoleID, err := ids.New()
	if err != nil {
		return onboardingmodels.Setup{}, fmt.Errorf("generate Admin role ID: %w", err)
	}
	developerRoleID, err := ids.New()
	if err != nil {
		return onboardingmodels.Setup{}, fmt.Errorf("generate Developer role ID: %w", err)
	}
	projectID, err := ids.New()
	if err != nil {
		return onboardingmodels.Setup{}, fmt.Errorf("generate project ID: %w", err)
	}
	sandboxID, err := ids.New()
	if err != nil {
		return onboardingmodels.Setup{}, fmt.Errorf("generate Sandbox ID: %w", err)
	}
	liveID, err := ids.New()
	if err != nil {
		return onboardingmodels.Setup{}, fmt.Errorf("generate Live ID: %w", err)
	}

	adminKey := organizationmodels.OrganizationRoleSystemKeyAdmin
	developerKey := organizationmodels.OrganizationRoleSystemKeyDeveloper
	organization := organizationmodels.Organization{
		ID: organizationID, OwnerUserID: userID, Name: request.OrganizationName,
	}
	roles := []organizationmodels.OrganizationRole{
		{ID: adminRoleID, OrganizationID: organizationID, Name: "Admin", SystemKey: &adminKey},
		{ID: developerRoleID, OrganizationID: organizationID, Name: "Developer", SystemKey: &developerKey},
	}
	membership := organizationmodels.OrganizationMembership{
		OrganizationID: organizationID,
		UserID:         userID,
		RoleID:         adminRoleID,
		Status:         organizationmodels.OrganizationMembershipStatusActive,
	}
	activatedAt := time.Now().UTC()
	project := projectmodels.Project{
		ID: projectID, OrganizationID: organizationID, Name: request.ProjectName,
		Slug: projectmodels.NewProjectSlug(request.ProjectName, projectID),
	}
	environments := []projectmodels.ProjectEnvironment{
		{ID: sandboxID, ProjectID: projectID, Name: projectmodels.ProjectEnvironmentSandbox, ActivatedAt: &activatedAt},
		{ID: liveID, ProjectID: projectID, Name: projectmodels.ProjectEnvironmentLive},
	}
	projectURL := service.clientBaseURL + "/dashboard/" + project.Slug

	return service.repository.SetupFirstProject(
		ctx, subjectID, organization, roles, membership, project, environments, projectURL,
	)
}
