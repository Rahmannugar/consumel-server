//go:build integration

package integration_test

import (
	"fmt"
	"testing"

	"github.com/Rahmannugar/consumel-server/internal/infra/database/testdb"
	"github.com/Rahmannugar/consumel-server/internal/organizations/repositories"
	organizationservices "github.com/Rahmannugar/consumel-server/internal/organizations/services"
	userrepositories "github.com/Rahmannugar/consumel-server/internal/users/repositories"
	userservices "github.com/Rahmannugar/consumel-server/internal/users/services"
	"github.com/google/uuid"
)

func TestActiveOrganizationAccessExcludesInactiveLifecycleStates(t *testing.T) {
	pool := testdb.OpenMigratedDatabase(t)
	userService := userservices.NewUserService(userrepositories.NewUserRepository(pool))
	user, err := userService.ResolveUser(t.Context(), "authlier-subject-access")
	if err != nil {
		t.Fatalf("resolve user: %v", err)
	}

	repository := repositories.NewOrganizationRepository(pool)
	service := organizationservices.NewOrganizationManagementService(repository)
	organizations := make([]uuid.UUID, 0, 7)
	roleIDs := make([]uuid.UUID, 0, 7)
	for index := range 7 {
		organization, membership, err := service.CreateOrganization(
			t.Context(),
			fmt.Sprintf("Organization %d", index+1),
			user.ID,
		)
		if err != nil {
			t.Fatalf("create organization %d: %v", index+1, err)
		}
		organizations = append(organizations, organization.ID)
		roleIDs = append(roleIDs, membership.RoleID)
	}

	statements := []struct {
		query string
		args  []any
	}{
		{"UPDATE organization_memberships SET removed_at = now() WHERE organization_id = $1 AND user_id = $2", []any{organizations[2], user.ID}},
		{"UPDATE organization_memberships SET status = 'suspended' WHERE organization_id = $1 AND user_id = $2", []any{organizations[3], user.ID}},
		{"UPDATE organizations SET deleted_at = now() WHERE id = $1", []any{organizations[4]}},
		{"UPDATE organizations SET suspended_at = now() WHERE id = $1", []any{organizations[5]}},
		{"UPDATE organization_roles SET deleted_at = now() WHERE id = $1", []any{roleIDs[6]}},
	}
	for _, statement := range statements {
		if _, err := pool.Exec(t.Context(), statement.query, statement.args...); err != nil {
			t.Fatalf("arrange inactive lifecycle state: %v", err)
		}
	}

	access, err := repository.ActiveOrganizationAccessByUser(t.Context(), user.ID)
	if err != nil {
		t.Fatalf("list active organization access: %v", err)
	}
	if len(access) != 2 {
		t.Fatalf("active organization count = %d, want 2", len(access))
	}
	if access[0].OrganizationID != organizations[0] || access[1].OrganizationID != organizations[1] {
		t.Fatalf("active organizations = %#v, want first two organizations", access)
	}
	for _, organizationAccess := range access {
		if !organizationAccess.Owner {
			t.Fatalf("owner access for %s was not preserved", organizationAccess.OrganizationID)
		}
	}
}
