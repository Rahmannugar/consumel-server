//go:build integration

package integration_test

import (
	"crypto/sha256"
	"errors"
	"sync"
	"testing"

	"github.com/Rahmannugar/consumel-server/internal/infra/database/testdb"
	organizationrepositories "github.com/Rahmannugar/consumel-server/internal/organizations/repositories"
	organizationservices "github.com/Rahmannugar/consumel-server/internal/organizations/services"
	projectmodels "github.com/Rahmannugar/consumel-server/internal/projects/models"
	projectrepositories "github.com/Rahmannugar/consumel-server/internal/projects/repositories"
	projectservices "github.com/Rahmannugar/consumel-server/internal/projects/services"
	userrepositories "github.com/Rahmannugar/consumel-server/internal/users/repositories"
	userservices "github.com/Rahmannugar/consumel-server/internal/users/services"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAPIKeyLifecycleStoresOnlyTheHashAndOneActiveKey(t *testing.T) {
	pool := testdb.OpenMigratedDatabase(t)
	userID, projectID, service := createAPIKeyProject(t, pool, "lifecycle")

	created, err := service.CreateAPIKey(
		t.Context(), userID, projectID, projectmodels.ProjectEnvironmentSandbox,
	)
	if err != nil {
		t.Fatalf("create API key: %v", err)
	}
	if created.Secret == "" || created.Prefix != "cm_test_" {
		t.Fatalf("created secret = %q with prefix %q", created.Secret, created.Prefix)
	}
	if created.LastFour != created.Secret[len(created.Secret)-4:] {
		t.Fatalf("last four = %q, want secret suffix", created.LastFour)
	}

	wantHash := sha256.Sum256([]byte(created.Secret))
	var storedHash []byte
	var storedSecretMatches int
	if err := pool.QueryRow(t.Context(), `
		SELECT key_hash,
		       (SELECT count(*) FROM project_api_keys WHERE key_hash = convert_to($2, 'UTF8'))
		FROM project_api_keys
		WHERE id = $1
	`, created.ID, created.Secret).Scan(&storedHash, &storedSecretMatches); err != nil {
		t.Fatalf("inspect stored API key: %v", err)
	}
	if string(storedHash) != string(wantHash[:]) {
		t.Fatal("stored API key hash does not match the plaintext digest")
	}
	if storedSecretMatches != 0 {
		t.Fatal("plaintext API key was stored as credential material")
	}

	if _, err := service.CreateAPIKey(
		t.Context(), userID, projectID, projectmodels.ProjectEnvironmentSandbox,
	); !errors.Is(err, projectmodels.ErrActiveAPIKeyExists) {
		t.Fatalf("second create error = %v, want active key exists", err)
	}

	replacement, err := service.ReplaceAPIKey(
		t.Context(), userID, projectID, projectmodels.ProjectEnvironmentSandbox,
	)
	if err != nil {
		t.Fatalf("replace API key: %v", err)
	}
	if replacement.Secret == created.Secret {
		t.Fatal("replacement reused the previous secret")
	}

	var activeCount, revokedCount int
	if err := pool.QueryRow(t.Context(), `
		SELECT count(*) FILTER (WHERE revoked_at IS NULL),
		       count(*) FILTER (WHERE revoked_at IS NOT NULL)
		FROM project_api_keys
	`).Scan(&activeCount, &revokedCount); err != nil {
		t.Fatalf("count API key lifecycle rows: %v", err)
	}
	if activeCount != 1 || revokedCount != 1 {
		t.Fatalf("API key counts = active:%d revoked:%d, want 1 and 1", activeCount, revokedCount)
	}

	if err := service.RevokeAPIKey(
		t.Context(), userID, projectID, projectmodels.ProjectEnvironmentSandbox,
	); err != nil {
		t.Fatalf("revoke API key: %v", err)
	}
	active, err := service.ActiveAPIKey(
		t.Context(), userID, projectID, projectmodels.ProjectEnvironmentSandbox,
	)
	if err != nil {
		t.Fatalf("load API key after revocation: %v", err)
	}
	if active != nil {
		t.Fatal("revoked API key is still active")
	}
}

func TestConcurrentAPIKeyCreationAllowsOneActiveKey(t *testing.T) {
	pool := testdb.OpenMigratedDatabase(t)
	userID, projectID, service := createAPIKeyProject(t, pool, "concurrent")

	start := make(chan struct{})
	errorsByAttempt := make(chan error, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			_, err := service.CreateAPIKey(
				t.Context(), userID, projectID, projectmodels.ProjectEnvironmentSandbox,
			)
			errorsByAttempt <- err
		}()
	}
	close(start)
	wait.Wait()
	close(errorsByAttempt)

	var successes, conflicts int
	for err := range errorsByAttempt {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, projectmodels.ErrActiveAPIKeyExists):
			conflicts++
		default:
			t.Fatalf("concurrent create returned unexpected error: %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrent outcomes = successes:%d conflicts:%d, want 1 and 1", successes, conflicts)
	}
}

func TestAPIKeyCreationRequiresAnActiveAccessibleEnvironment(t *testing.T) {
	pool := testdb.OpenMigratedDatabase(t)
	ownerID, projectID, service := createAPIKeyProject(t, pool, "authorization")

	if _, err := service.CreateAPIKey(
		t.Context(), ownerID, projectID, projectmodels.ProjectEnvironmentLive,
	); !errors.Is(err, projectmodels.ErrProjectEnvironmentInactive) {
		t.Fatalf("inactive Live create error = %v, want environment inactive", err)
	}
	activated, err := service.ActivateEnvironment(
		t.Context(), ownerID, projectID, projectmodels.ProjectEnvironmentLive,
	)
	if err != nil {
		t.Fatalf("activate Live: %v", err)
	}
	if activated.ActivatedAt == nil {
		t.Fatal("Live remains inactive after explicit activation")
	}
	if _, err := service.CreateAPIKey(
		t.Context(), ownerID, projectID, projectmodels.ProjectEnvironmentLive,
	); err != nil {
		t.Fatalf("create Live API key after activation: %v", err)
	}

	otherUserRepository := userrepositories.NewUserRepository(pool)
	otherUserService := userservices.NewUserService(otherUserRepository)
	otherUser, err := otherUserService.ResolveUser(t.Context(), "api_key_other_user")
	if err != nil {
		t.Fatalf("create other user: %v", err)
	}
	if _, err := service.ActiveAPIKey(
		t.Context(), otherUser.ID, projectID, projectmodels.ProjectEnvironmentSandbox,
	); !errors.Is(err, projectmodels.ErrProjectEnvironmentUnavailable) {
		t.Fatalf("cross-tenant read error = %v, want environment unavailable", err)
	}
}

func createAPIKeyProject(
	t *testing.T,
	pool *pgxpool.Pool,
	suffix string,
) (uuid.UUID, uuid.UUID, *projectservices.ProjectManagementService) {
	t.Helper()
	userService := userservices.NewUserService(userrepositories.NewUserRepository(pool))
	user, err := userService.ResolveUser(t.Context(), "api_key_owner_"+suffix)
	if err != nil {
		t.Fatalf("create API key owner: %v", err)
	}
	organizationService := organizationservices.NewOrganizationManagementService(
		organizationrepositories.NewOrganizationRepository(pool),
	)
	organization, _, err := organizationService.CreateOrganization(
		t.Context(), "API Key Organization "+suffix, user.ID,
	)
	if err != nil {
		t.Fatalf("create API key organization: %v", err)
	}
	service := projectservices.NewProjectManagementService(
		projectrepositories.NewProjectRepository(pool),
	)
	project, err := service.CreateProject(
		t.Context(), organization.ID, "API Key Project "+suffix,
	)
	if err != nil {
		t.Fatalf("create API key project: %v", err)
	}
	return user.ID, project.ID, service
}
