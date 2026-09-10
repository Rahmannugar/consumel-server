//go:build integration

package integration_test

import (
	"testing"

	"github.com/Rahmannugar/consumel-server/internal/infra/database/testdb"
	organizationrepositories "github.com/Rahmannugar/consumel-server/internal/organizations/repositories"
	organizationservices "github.com/Rahmannugar/consumel-server/internal/organizations/services"
	projectmodels "github.com/Rahmannugar/consumel-server/internal/projects/models"
	projectrepositories "github.com/Rahmannugar/consumel-server/internal/projects/repositories"
	projectservices "github.com/Rahmannugar/consumel-server/internal/projects/services"
	userrepositories "github.com/Rahmannugar/consumel-server/internal/users/repositories"
	userservices "github.com/Rahmannugar/consumel-server/internal/users/services"
)

func TestProjectCreationPersistsSandboxAndLiveEnvironments(t *testing.T) {
	pool := testdb.OpenMigratedDatabase(t)

	userRepository := userrepositories.NewUserRepository(pool)
	userService := userservices.NewUserService(userRepository)
	user, err := userService.CreateUser(t.Context(), "user_project_owner")
	if err != nil {
		t.Fatalf("create owner user: %v", err)
	}
	organizationRepository := organizationrepositories.NewOrganizationRepository(pool)
	organizationService := organizationservices.NewOrganizationManagementService(organizationRepository)
	organization, _, err := organizationService.CreateOrganization(
		t.Context(),
		"Project Test Organization",
		user.ID,
	)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}

	projectRepository := projectrepositories.NewProjectRepository(pool)
	projectService := projectservices.NewProjectManagementService(projectRepository)
	project, err := projectService.CreateProject(t.Context(), organization.ID, "Consumel Test Project")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if project.ID.Version() != 7 {
		t.Fatalf("project ID version = %d, want 7", project.ID.Version())
	}

	environments, err := projectRepository.ProjectEnvironments(t.Context(), project.ID)
	if err != nil {
		t.Fatalf("list project environments: %v", err)
	}
	if len(environments) != 2 {
		t.Fatalf("environment count = %d, want 2", len(environments))
	}

	byName := make(map[projectmodels.ProjectEnvironmentName]projectmodels.ProjectEnvironment, len(environments))
	for _, environment := range environments {
		byName[environment.Name] = environment
		if environment.ID.Version() != 7 {
			t.Fatalf("%s environment ID version = %d, want 7", environment.Name, environment.ID.Version())
		}
	}

	sandbox, exists := byName[projectmodels.ProjectEnvironmentSandbox]
	if !exists {
		t.Fatal("sandbox environment was not persisted")
	}
	if sandbox.ActivatedAt == nil {
		t.Fatal("sandbox environment is not active")
	}

	live, exists := byName[projectmodels.ProjectEnvironmentLive]
	if !exists {
		t.Fatal("live environment was not persisted")
	}
	if live.ActivatedAt != nil {
		t.Fatal("live environment is active before activation")
	}
}
