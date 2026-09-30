//go:build integration

package integration_test

import (
	"errors"
	"fmt"
	"testing"

	customermodels "github.com/Rahmannugar/consumel-server/internal/customers/models"
	customerrepositories "github.com/Rahmannugar/consumel-server/internal/customers/repositories"
	customerservices "github.com/Rahmannugar/consumel-server/internal/customers/services"
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

func TestCustomersRemainIsolatedByProjectEnvironment(t *testing.T) {
	pool := testdb.OpenMigratedDatabase(t)
	_, project, projectService := createCustomerProject(t, pool)
	sandbox := environmentID(t, project, projectmodels.ProjectEnvironmentSandbox)
	live := environmentID(t, project, projectmodels.ProjectEnvironmentLive)
	service := customerservices.NewCustomerService(customerrepositories.NewCustomerRepository(pool))
	input := customermodels.CreateCustomerRequest{CustomerID: "user_123", Name: text("Jordan")}

	sandboxCustomer, err := service.Create(t.Context(), sandbox, input)
	if err != nil {
		t.Fatalf("create Sandbox customer: %v", err)
	}
	if _, err := service.Create(t.Context(), sandbox, input); !errors.Is(err, customermodels.ErrCustomerExists) {
		t.Fatalf("duplicate Sandbox customer error = %v, want conflict", err)
	}
	if _, err := service.Create(t.Context(), live, input); err != nil {
		t.Fatalf("create same customer ID in Live: %v", err)
	}
	loaded, err := service.Get(t.Context(), sandbox, "user_123")
	if err != nil {
		t.Fatalf("get Sandbox customer: %v", err)
	}
	if loaded.ID != sandboxCustomer.ID {
		t.Fatalf("loaded customer ID = %s, want %s", loaded.ID, sandboxCustomer.ID)
	}

	updated, err := service.Update(t.Context(), sandbox, "user_123", customermodels.UpdateCustomerRequest{
		Email: text("jordan@example.com"), Metadata: customermodels.Metadata{Plan: text("growth")},
	})
	if err != nil {
		t.Fatalf("update Sandbox customer: %v", err)
	}
	if updated.Name != nil || updated.Email == nil || *updated.Email != "jordan@example.com" {
		t.Fatalf("updated customer = %#v", updated)
	}

	_, _ = projectService, project
}

func TestCustomerListUsesStableCursorPagination(t *testing.T) {
	pool := testdb.OpenMigratedDatabase(t)
	_, project, _ := createCustomerProject(t, pool)
	sandbox := environmentID(t, project, projectmodels.ProjectEnvironmentSandbox)
	service := customerservices.NewCustomerService(customerrepositories.NewCustomerRepository(pool))
	for index := range 3 {
		request := customermodels.CreateCustomerRequest{
			CustomerID: fmt.Sprintf("customer_%d", index),
		}
		if index == 1 {
			request.Name = text("CX Northwind Labs")
			request.Email = text("billing@northwind.example")
		}
		_, err := service.Create(t.Context(), sandbox, request)
		if err != nil {
			t.Fatalf("create customer %d: %v", index, err)
		}
	}
	first, cursor, err := service.List(t.Context(), sandbox, nil, 2, "")
	if err != nil {
		t.Fatalf("list first page: %v", err)
	}
	if len(first) != 2 || cursor == nil {
		t.Fatalf("first page has %d customers and cursor %v", len(first), cursor)
	}
	second, next, err := service.List(t.Context(), sandbox, cursor, 2, "")
	if err != nil {
		t.Fatalf("list second page: %v", err)
	}
	if len(second) != 1 || next != nil {
		t.Fatalf("second page has %d customers and cursor %v", len(second), next)
	}
	byName, _, err := service.List(t.Context(), sandbox, nil, 10, "northwind")
	if err != nil {
		t.Fatalf("search customers by name: %v", err)
	}
	if len(byName) != 1 || byName[0].CustomerID != "customer_1" {
		t.Fatalf("name search returned %#v", byName)
	}
	byShortTerm, _, err := service.List(t.Context(), sandbox, nil, 10, "CX")
	if err != nil {
		t.Fatalf("search customers by short full-text term: %v", err)
	}
	if len(byShortTerm) != 1 || byShortTerm[0].CustomerID != "customer_1" {
		t.Fatalf("short full-text search returned %#v", byShortTerm)
	}
	byPartialTerm, _, err := service.List(t.Context(), sandbox, nil, 10, "orthw")
	if err != nil {
		t.Fatalf("search customers by partial term: %v", err)
	}
	if len(byPartialTerm) != 1 || byPartialTerm[0].CustomerID != "customer_1" {
		t.Fatalf("partial trigram search returned %#v", byPartialTerm)
	}
	byEmail, _, err := service.List(t.Context(), sandbox, nil, 10, "BILLING@NORTHWIND")
	if err != nil {
		t.Fatalf("search customers by email: %v", err)
	}
	if len(byEmail) != 1 || byEmail[0].CustomerID != "customer_1" {
		t.Fatalf("email search returned %#v", byEmail)
	}
}

func createCustomerProject(
	t *testing.T,
	pool *pgxpool.Pool,
) (uuid.UUID, projectmodels.Project, *projectservices.ProjectManagementService) {
	t.Helper()
	user, err := userservices.NewUserService(userrepositories.NewUserRepository(pool)).ResolveUser(
		t.Context(), "customer_test_owner",
	)
	if err != nil {
		t.Fatalf("create owner: %v", err)
	}
	organization, _, err := organizationservices.NewOrganizationManagementService(
		organizationrepositories.NewOrganizationRepository(pool),
	).CreateOrganization(t.Context(), "Customer Test Organization", user.ID)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	projectService := projectservices.NewProjectManagementService(
		projectrepositories.NewProjectRepository(pool),
	)
	project, err := projectService.CreateProject(t.Context(), organization.ID, "Customer Test Project")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	return user.ID, project, projectService
}

func environmentID(
	t *testing.T,
	project projectmodels.Project,
	name projectmodels.ProjectEnvironmentName,
) uuid.UUID {
	t.Helper()
	for _, environment := range project.Environments {
		if environment.Name == name {
			return environment.ID
		}
	}
	t.Fatalf("%s environment not found", name)
	return uuid.Nil
}

func text(value string) *string {
	return &value
}
