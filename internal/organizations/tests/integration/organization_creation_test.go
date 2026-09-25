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
)

func TestOrganizationCreationPersistsBuiltInRoles(t *testing.T) {
	pool := testdb.OpenMigratedDatabase(t)

	userRepository := userrepositories.NewUserRepository(pool)
	userService := userservices.NewUserService(userRepository)
	user, err := userService.ResolveUser(t.Context(), "user_organization_owner")
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

	roles, err := repository.OrganizationRoles(t.Context(), organization.ID)
	if err != nil {
		t.Fatalf("list organization roles: %v", err)
	}
	if len(roles) != 2 {
		t.Fatalf("role count = %d, want 2", len(roles))
	}

	rolesBySystemKey := make(map[organizationmodels.OrganizationRoleSystemKey]organizationmodels.OrganizationRole, len(roles))
	for _, role := range roles {
		if role.SystemKey == nil {
			t.Fatalf("built-in role %q has no system key", role.Name)
		}
		rolesBySystemKey[*role.SystemKey] = role
	}

	adminRole, hasAdmin := rolesBySystemKey[organizationmodels.OrganizationRoleSystemKeyAdmin]
	_, hasDeveloper := rolesBySystemKey[organizationmodels.OrganizationRoleSystemKeyDeveloper]
	if !hasAdmin || !hasDeveloper {
		t.Fatalf("built-in roles = %#v, want Admin and Developer", rolesBySystemKey)
	}
	if membership.RoleID != adminRole.ID {
		t.Fatalf("owner membership role ID = %s, want Admin role %s", membership.RoleID, adminRole.ID)
	}
}
