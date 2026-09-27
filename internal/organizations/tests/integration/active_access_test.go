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
	repository := repositories.NewOrganizationRepository(pool)
	service := organizationservices.NewOrganizationManagementService(repository)
	users := make([]uuid.UUID, 0, 6)
	organizations := make([]uuid.UUID, 0, 6)
	roleIDs := make([]uuid.UUID, 0, 6)
	for index := range 6 {
		user, err := userService.ResolveUser(t.Context(), fmt.Sprintf("authlier-subject-access-%d", index+1))
		if err != nil {
			t.Fatalf("resolve user %d: %v", index+1, err)
		}
		organization, membership, err := service.CreateOrganization(
			t.Context(),
			fmt.Sprintf("Organization %d", index+1),
			user.ID,
		)
		if err != nil {
			t.Fatalf("create organization %d: %v", index+1, err)
		}
		users = append(users, user.ID)
		organizations = append(organizations, organization.ID)
		roleIDs = append(roleIDs, membership.RoleID)
	}

	statements := []struct {
		query string
		args  []any
	}{
		{"UPDATE organization_memberships SET removed_at = now() WHERE organization_id = $1 AND user_id = $2", []any{organizations[1], users[1]}},
		{"UPDATE organization_memberships SET status = 'suspended' WHERE organization_id = $1 AND user_id = $2", []any{organizations[2], users[2]}},
		{"UPDATE organizations SET deleted_at = now() WHERE id = $1", []any{organizations[3]}},
		{"UPDATE organizations SET suspended_at = now() WHERE id = $1", []any{organizations[4]}},
		{"UPDATE organization_roles SET deleted_at = now() WHERE id = $1", []any{roleIDs[5]}},
	}
	for _, statement := range statements {
		if _, err := pool.Exec(t.Context(), statement.query, statement.args...); err != nil {
			t.Fatalf("arrange inactive lifecycle state: %v", err)
		}
	}

	access, err := repository.ActiveOrganizationAccessByUser(t.Context(), users[0])
	if err != nil {
		t.Fatalf("list active organization access: %v", err)
	}
	if len(access) != 1 {
		t.Fatalf("active organization count = %d, want 1", len(access))
	}
	if access[0].OrganizationID != organizations[0] {
		t.Fatalf("active organization = %#v, want first organization", access)
	}
	for _, organizationAccess := range access {
		if !organizationAccess.Owner {
			t.Fatalf("owner access for %s was not preserved", organizationAccess.OrganizationID)
		}
	}

	for index := 1; index < len(users); index++ {
		inactiveAccess, err := repository.ActiveOrganizationAccessByUser(t.Context(), users[index])
		if err != nil {
			t.Fatalf("list inactive organization access %d: %v", index, err)
		}
		if len(inactiveAccess) != 0 {
			t.Fatalf("inactive organization access %d = %#v, want none", index, inactiveAccess)
		}
	}
}
