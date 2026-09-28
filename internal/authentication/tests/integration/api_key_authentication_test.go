//go:build integration

package integration_test

import (
	"errors"
	"testing"

	authenticationrepositories "github.com/Rahmannugar/consumel-server/internal/authentication/repositories"
	authenticationservices "github.com/Rahmannugar/consumel-server/internal/authentication/services"
	"github.com/Rahmannugar/consumel-server/internal/infra/database/testdb"
	organizationrepositories "github.com/Rahmannugar/consumel-server/internal/organizations/repositories"
	organizationservices "github.com/Rahmannugar/consumel-server/internal/organizations/services"
	projectmodels "github.com/Rahmannugar/consumel-server/internal/projects/models"
	projectrepositories "github.com/Rahmannugar/consumel-server/internal/projects/repositories"
	projectservices "github.com/Rahmannugar/consumel-server/internal/projects/services"
	userrepositories "github.com/Rahmannugar/consumel-server/internal/users/repositories"
	userservices "github.com/Rahmannugar/consumel-server/internal/users/services"
)

func TestProjectAPIKeyAuthenticationResolvesActiveEnvironmentAndRejectsReplacement(t *testing.T) {
	pool := testdb.OpenMigratedDatabase(t)
	user, err := userservices.NewUserService(userrepositories.NewUserRepository(pool)).ResolveUser(
		t.Context(), "api_key_authentication_owner",
	)
	if err != nil {
		t.Fatalf("create owner: %v", err)
	}
	organization, _, err := organizationservices.NewOrganizationManagementService(
		organizationrepositories.NewOrganizationRepository(pool),
	).CreateOrganization(t.Context(), "API Authentication Organization", user.ID)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	projects := projectservices.NewProjectManagementService(
		projectrepositories.NewProjectRepository(pool),
	)
	project, err := projects.CreateProject(t.Context(), organization.ID, "API Authentication Project")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	created, err := projects.CreateAPIKey(
		t.Context(), user.ID, project.ID, projectmodels.ProjectEnvironmentSandbox,
	)
	if err != nil {
		t.Fatalf("create API key: %v", err)
	}
	authenticator := authenticationservices.NewAPIKeyAuthenticator(
		authenticationrepositories.NewAPIKeyContextRepository(pool),
	)

	resolved, err := authenticator.Authenticate(t.Context(), "Bearer "+created.Secret)
	if err != nil {
		t.Fatalf("authenticate created key: %v", err)
	}
	if resolved.OrganizationID != organization.ID || resolved.ProjectID != project.ID ||
		resolved.ProjectEnvironmentID != created.ProjectEnvironmentID || resolved.Environment != "sandbox" {
		t.Fatalf("resolved context = %#v", resolved)
	}

	replacement, err := projects.ReplaceAPIKey(
		t.Context(), user.ID, project.ID, projectmodels.ProjectEnvironmentSandbox,
	)
	if err != nil {
		t.Fatalf("replace API key: %v", err)
	}
	if _, err := authenticator.Authenticate(
		t.Context(), "Bearer "+created.Secret,
	); !errors.Is(err, authenticationservices.ErrInvalidAPIKey) {
		t.Fatalf("replaced key error = %v, want invalid API key", err)
	}
	if _, err := authenticator.Authenticate(
		t.Context(), "Bearer "+replacement.Secret,
	); err != nil {
		t.Fatalf("authenticate replacement: %v", err)
	}

	var lastUsedPresent bool
	if err := pool.QueryRow(t.Context(), `
		SELECT last_used_at IS NOT NULL
		FROM project_api_keys
		WHERE id = $1
	`, replacement.ID).Scan(&lastUsedPresent); err != nil {
		t.Fatalf("inspect last used timestamp: %v", err)
	}
	if !lastUsedPresent {
		t.Fatal("successful API key authentication did not record last use")
	}
}
