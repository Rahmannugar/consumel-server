//go:build integration

package integration_test

import (
	"testing"

	"github.com/Rahmannugar/consumel-server/internal/infra/database/testdb"
	organizationmodels "github.com/Rahmannugar/consumel-server/internal/organizations/models"
	"github.com/Rahmannugar/consumel-server/internal/organizations/repositories"
	organizationservices "github.com/Rahmannugar/consumel-server/internal/organizations/services"
	userrepositories "github.com/Rahmannugar/consumel-server/internal/users/repositories"
	userservices "github.com/Rahmannugar/consumel-server/internal/users/services"
	"github.com/google/uuid"
)

func TestOrganizationPersistence(t *testing.T) {
	pool := testdb.OpenMigratedDatabase(t)

	userRepository := userrepositories.NewUserRepository(pool)
	userService := userservices.NewUserService(userRepository)
	user, err := userService.CreateUser(t.Context(), "user_organization_owner")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	repository := repositories.NewOrganizationRepository(pool)
	service := organizationservices.NewOrganizationManagementService(repository)
	organization, membership, err := service.CreateOrganization(
		t.Context(),
		"Consumel Test Organization",
		user.ID,
	)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	if organization.ID.Version() != 7 {
		t.Fatalf("organization ID version = %d, want 7", organization.ID.Version())
	}
	if organization.OwnerUserID != user.ID {
		t.Fatalf("owner user ID = %s, want %s", organization.OwnerUserID, user.ID)
	}
	if membership.Role != organizationmodels.OrganizationMembershipRoleAdmin {
		t.Fatalf("membership role = %q, want %q", membership.Role, organizationmodels.OrganizationMembershipRoleAdmin)
	}
	if membership.Status != organizationmodels.OrganizationMembershipStatusActive {
		t.Fatalf("membership status = %q, want %q", membership.Status, organizationmodels.OrganizationMembershipStatusActive)
	}
	if membership.RemovedAt != nil {
		t.Fatal("owner membership is removed at creation")
	}
	if organization.DeletedAt != nil {
		t.Fatal("organization is deleted at creation")
	}
	if organization.SuspendedAt != nil {
		t.Fatal("organization is suspended at creation")
	}

	storedMembership, err := repository.OrganizationMembership(t.Context(), organization.ID, user.ID)
	if err != nil {
		t.Fatalf("get owner membership: %v", err)
	}
	if storedMembership != membership {
		t.Fatalf("stored membership = %#v, want %#v", storedMembership, membership)
	}

	missingUserID := uuid.Must(uuid.NewV7())
	_, _, err = service.CreateOrganization(t.Context(), "Must Roll Back", missingUserID)
	if err == nil {
		t.Fatal("create organization with missing owner succeeded")
	}

	var rolledBackOrganizations int
	if err := pool.QueryRow(
		t.Context(),
		"SELECT count(*) FROM organizations WHERE name = $1",
		"Must Roll Back",
	).Scan(&rolledBackOrganizations); err != nil {
		t.Fatalf("count rolled-back organizations: %v", err)
	}
	if rolledBackOrganizations != 0 {
		t.Fatalf("rolled-back organization count = %d, want 0", rolledBackOrganizations)
	}
}
