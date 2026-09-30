//go:build integration

package integration_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Rahmannugar/consumel-server/internal/infra/database/testdb"
	metermodels "github.com/Rahmannugar/consumel-server/internal/meters/models"
	meterrepositories "github.com/Rahmannugar/consumel-server/internal/meters/repositories"
	meterservices "github.com/Rahmannugar/consumel-server/internal/meters/services"
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

func TestMetersShareOneProjectDefinitionAcrossEnvironments(t *testing.T) {
	pool := testdb.OpenMigratedDatabase(t)
	project := createMeterProject(t, pool)
	sandbox := environmentID(t, project, projectmodels.ProjectEnvironmentSandbox)
	live := environmentID(t, project, projectmodels.ProjectEnvironmentLive)
	service := meterservices.NewMeterService(meterrepositories.NewMeterRepository(pool))
	description := "Requests processed by the API."

	sandboxMeter, err := service.Create(t.Context(), sandbox, metermodels.CreateMeterRequest{
		MeterKey: "api_calls", Name: "API calls", Description: &description,
		Type: metermodels.MeterTypePostpaid,
	})
	if err != nil {
		t.Fatalf("create Sandbox meter: %v", err)
	}
	if _, err := service.Create(t.Context(), sandbox, metermodels.CreateMeterRequest{
		MeterKey: "api_calls", Name: "Duplicate", Type: metermodels.MeterTypePrepaid,
	}); !errors.Is(err, metermodels.ErrMeterDefinitionConflict) {
		t.Fatalf("changed Sandbox definition error = %v, want definition conflict", err)
	}
	liveMeter, err := service.Create(t.Context(), live, metermodels.CreateMeterRequest{
		MeterKey: "api_calls", Name: "API calls", Description: &description,
		Type: metermodels.MeterTypePostpaid,
	})
	if err != nil {
		t.Fatalf("create Live meter: %v", err)
	}
	if liveMeter.ID != sandboxMeter.ID {
		t.Fatalf("Live identity = %s, want project identity %s", liveMeter.ID, sandboxMeter.ID)
	}
	loaded, err := service.Get(t.Context(), sandbox, "api_calls")
	if err != nil {
		t.Fatalf("get Sandbox meter: %v", err)
	}
	if loaded.Name != "API calls" || loaded.Type != metermodels.MeterTypePostpaid {
		t.Fatalf("project meter definition = %#v", loaded)
	}
	if _, err := service.Create(t.Context(), live, metermodels.CreateMeterRequest{
		MeterKey: "api_calls", Name: "API calls", Description: &description,
		Type: metermodels.MeterTypePostpaid,
	}); !errors.Is(err, metermodels.ErrMeterExists) {
		t.Fatalf("duplicate Live meter error = %v, want environment conflict", err)
	}
}

func TestMeterListsUseStableCursorAndExcludeArchivedConfigurations(t *testing.T) {
	pool := testdb.OpenMigratedDatabase(t)
	project := createMeterProject(t, pool)
	sandbox := environmentID(t, project, projectmodels.ProjectEnvironmentSandbox)
	service := meterservices.NewMeterService(meterrepositories.NewMeterRepository(pool))
	created := make([]metermodels.Meter, 0, 3)
	for index := range 3 {
		name := fmt.Sprintf("API calls %d", index)
		if index == 1 {
			name = "AI"
		}
		meter, err := service.Create(t.Context(), sandbox, metermodels.CreateMeterRequest{
			MeterKey: fmt.Sprintf("api_calls_%d", index), Name: name,
			Type: metermodels.MeterTypePostpaid,
		})
		if err != nil {
			t.Fatalf("create meter %d: %v", index, err)
		}
		created = append(created, meter)
	}
	first, cursor, err := service.List(t.Context(), sandbox, nil, 2, "")
	if err != nil {
		t.Fatalf("list first page: %v", err)
	}
	if len(first) != 2 || cursor == nil {
		t.Fatalf("first page has %d meters and cursor %v", len(first), cursor)
	}
	second, next, err := service.List(t.Context(), sandbox, cursor, 2, "")
	if err != nil {
		t.Fatalf("list second page: %v", err)
	}
	if len(second) != 1 || next != nil {
		t.Fatalf("second page has %d meters and cursor %v", len(second), next)
	}
	if _, err := pool.Exec(t.Context(), `
		UPDATE project_environment_meters SET archived_at = now()
		WHERE project_environment_id = $1 AND meter_id = $2
	`, sandbox, created[0].ID); err != nil {
		t.Fatalf("archive meter: %v", err)
	}
	if _, err := service.Get(t.Context(), sandbox, created[0].MeterKey); !errors.Is(err, metermodels.ErrMeterNotFound) {
		t.Fatalf("archived meter get error = %v, want not found", err)
	}
	remaining, _, err := service.List(t.Context(), sandbox, nil, 10, "")
	if err != nil {
		t.Fatalf("list after archive: %v", err)
	}
	if len(remaining) != 2 {
		t.Fatalf("active meter count = %d, want 2", len(remaining))
	}
	matching, _, err := service.List(t.Context(), sandbox, nil, 10, "CALLS_1")
	if err != nil {
		t.Fatalf("search meters: %v", err)
	}
	if len(matching) != 1 || matching[0].MeterKey != "api_calls_1" {
		t.Fatalf("meter search returned %#v", matching)
	}
	shortTerm, _, err := service.List(t.Context(), sandbox, nil, 10, "AI")
	if err != nil {
		t.Fatalf("search meters by short full-text term: %v", err)
	}
	if len(shortTerm) != 1 || shortTerm[0].MeterKey != "api_calls_1" {
		t.Fatalf("short full-text meter search returned %#v", shortTerm)
	}
	archived, _, err := service.List(t.Context(), sandbox, nil, 10, "API calls 0")
	if err != nil {
		t.Fatalf("search archived meter: %v", err)
	}
	if len(archived) != 0 {
		t.Fatalf("archived meter search returned %#v", archived)
	}
}

func createMeterProject(t *testing.T, pool *pgxpool.Pool) projectmodels.Project {
	t.Helper()
	user, err := userservices.NewUserService(userrepositories.NewUserRepository(pool)).ResolveUser(t.Context(), "meter_test_owner")
	if err != nil {
		t.Fatalf("create owner: %v", err)
	}
	organization, _, err := organizationservices.NewOrganizationManagementService(
		organizationrepositories.NewOrganizationRepository(pool),
	).CreateOrganization(t.Context(), "Meter Test Organization", user.ID)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := projectservices.NewProjectManagementService(
		projectrepositories.NewProjectRepository(pool),
	).CreateProject(t.Context(), organization.ID, "Meter Test Project")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	return project
}

func environmentID(t *testing.T, project projectmodels.Project, name projectmodels.ProjectEnvironmentName) uuid.UUID {
	t.Helper()
	for _, environment := range project.Environments {
		if environment.Name == name {
			return environment.ID
		}
	}
	t.Fatalf("%s environment not found", name)
	return uuid.Nil
}
