//go:build integration

package integration_test

import (
	"errors"
	"testing"
	"time"

	consumptionmodels "github.com/Rahmannugar/consumel-server/internal/core/consumption/models"
	consumptionrepositories "github.com/Rahmannugar/consumel-server/internal/core/consumption/repositories"
	consumptionservices "github.com/Rahmannugar/consumel-server/internal/core/consumption/services"
	customermodels "github.com/Rahmannugar/consumel-server/internal/customers/models"
	customerrepositories "github.com/Rahmannugar/consumel-server/internal/customers/repositories"
	customerservices "github.com/Rahmannugar/consumel-server/internal/customers/services"
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

func TestPortfolioAggregatesAccessibleProjectsWithoutMixingEnvironments(t *testing.T) {
	pool := testdb.OpenMigratedDatabase(t)
	user, err := userservices.NewUserService(
		userrepositories.NewUserRepository(pool),
	).ResolveUser(t.Context(), "portfolio_owner")
	if err != nil {
		t.Fatalf("create portfolio owner: %v", err)
	}
	organization, _, err := organizationservices.NewOrganizationManagementService(
		organizationrepositories.NewOrganizationRepository(pool),
	).CreateOrganization(t.Context(), "Portfolio Organization", user.ID)
	if err != nil {
		t.Fatalf("create portfolio organization: %v", err)
	}

	repository := projectrepositories.NewProjectRepository(pool)
	projectService := projectservices.NewProjectManagementService(repository)
	first, err := projectService.CreateProject(t.Context(), organization.ID, "First API")
	if err != nil {
		t.Fatalf("create first project: %v", err)
	}
	second, err := projectService.CreateProject(t.Context(), organization.ID, "Second API")
	if err != nil {
		t.Fatalf("create second project: %v", err)
	}

	firstEnvironment := sandboxEnvironment(t, first)
	secondEnvironment := sandboxEnvironment(t, second)
	recordPortfolioUsage(t, pool, user.ID, firstEnvironment.ID, "first_customer", true)
	recordPortfolioUsage(t, pool, user.ID, secondEnvironment.ID, "second_customer", false)

	now := time.Now().UTC()
	portfolioService := projectservices.NewPortfolioService(repository)
	portfolio, err := portfolioService.Get(
		t.Context(), user.ID,
		projectmodels.PortfolioFilter{
			Environment: projectmodels.ProjectEnvironmentSandbox,
			Interval:    projectmodels.PortfolioIntervalHour,
		},
		now.Add(-time.Hour).Format(time.RFC3339),
		now.Add(time.Minute).Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("get Sandbox portfolio: %v", err)
	}
	if len(portfolio.Projects) != 2 || portfolio.Summary.AllowedOperations != 2 || portfolio.Summary.BlockedOperations != 1 {
		t.Fatalf("Sandbox portfolio = %#v", portfolio)
	}

	live, err := portfolioService.Get(
		t.Context(), user.ID,
		projectmodels.PortfolioFilter{
			Environment: projectmodels.ProjectEnvironmentLive,
			Interval:    projectmodels.PortfolioIntervalHour,
		},
		now.Add(-time.Hour).Format(time.RFC3339),
		now.Add(time.Minute).Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("get Live portfolio: %v", err)
	}
	if len(live.Projects) != 2 || live.Summary.AllowedOperations != 0 || live.Summary.BlockedOperations != 0 {
		t.Fatalf("Live portfolio = %#v", live)
	}
	for _, project := range live.Projects {
		if project.ActivatedAt != nil {
			t.Fatalf("Live project %s unexpectedly active", project.Name)
		}
	}
}

func sandboxEnvironment(t *testing.T, project projectmodels.Project) projectmodels.ProjectEnvironment {
	t.Helper()
	for _, environment := range project.Environments {
		if environment.Name == projectmodels.ProjectEnvironmentSandbox {
			return environment
		}
	}
	t.Fatal("project has no Sandbox environment")
	return projectmodels.ProjectEnvironment{}
}

func recordPortfolioUsage(
	t *testing.T,
	pool *pgxpool.Pool,
	actorID uuid.UUID,
	environmentID uuid.UUID,
	customerID string,
	includeBlocked bool,
) {
	t.Helper()
	customerService := customerservices.NewCustomerService(customerrepositories.NewCustomerRepository(pool))
	if _, err := customerService.Create(t.Context(), environmentID, customermodels.CreateCustomerRequest{CustomerID: customerID}); err != nil {
		t.Fatalf("create portfolio customer: %v", err)
	}
	meterService := meterservices.NewMeterService(meterrepositories.NewMeterRepository(pool))
	if _, err := meterService.Create(t.Context(), environmentID, metermodels.CreateMeterRequest{
		MeterKey: "api_calls", Name: "API calls", Type: metermodels.MeterTypePrepaid,
	}); err != nil {
		t.Fatalf("create portfolio meter: %v", err)
	}
	balanceService := consumptionservices.NewBalanceService(consumptionrepositories.NewBalanceRepository(pool))
	if _, _, err := balanceService.Add(
		t.Context(), environmentID,
		consumptionmodels.BalanceMutationSource{Type: consumptionmodels.BalanceSourceDashboardUser, ActorID: actorID},
		portfolioV7(t).String(),
		consumptionmodels.AddBalanceRequest{CustomerID: customerID, MeterKey: "api_calls", Quantity: 2},
	); err != nil {
		t.Fatalf("add portfolio balance: %v", err)
	}
	consumeService := consumptionservices.NewConsumeService(consumptionrepositories.NewConsumeRepository(pool))
	if _, _, err := consumeService.Consume(t.Context(), environmentID, uuid.Nil, portfolioV7(t).String(), consumptionmodels.ConsumeRequest{
		CustomerID: customerID, MeterKey: "api_calls", Quantity: 1,
	}); err != nil {
		t.Fatalf("record allowed portfolio usage: %v", err)
	}
	if includeBlocked {
		_, _, err := consumeService.Consume(t.Context(), environmentID, uuid.Nil, portfolioV7(t).String(), consumptionmodels.ConsumeRequest{
			CustomerID: customerID, MeterKey: "api_calls", Quantity: 5,
		})
		if !errors.Is(err, consumptionmodels.ErrInsufficientBalance) {
			t.Fatalf("record blocked portfolio usage: %v", err)
		}
	}
}

func portfolioV7(t *testing.T) uuid.UUID {
	t.Helper()
	value, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("generate UUIDv7: %v", err)
	}
	return value
}
