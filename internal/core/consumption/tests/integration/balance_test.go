//go:build integration

package integration_test

import (
	"errors"
	"sync"
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

func TestBalancesPreserveIdempotencyConcurrencyAndEnvironmentIsolation(t *testing.T) {
	pool := testdb.OpenMigratedDatabase(t)
	fixture := createBalanceFixture(t, pool)
	service := consumptionservices.NewBalanceService(consumptionrepositories.NewBalanceRepository(pool))
	ctx := t.Context()

	key := newV7(t)
	first, replayed, err := service.Add(t.Context(), fixture.sandboxID, key.String(), consumptionmodels.AddBalanceRequest{
		CustomerID: fixture.customerID, MeterKey: fixture.meterKey, Quantity: 10,
	})
	if err != nil || replayed {
		t.Fatalf("first addition = %#v, replayed %t, error %v", first, replayed, err)
	}
	replay, replayed, err := service.Add(t.Context(), fixture.sandboxID, key.String(), consumptionmodels.AddBalanceRequest{
		CustomerID: fixture.customerID, MeterKey: fixture.meterKey, Quantity: 10,
	})
	if err != nil || !replayed {
		t.Fatalf("replayed addition = %#v, replayed %t, error %v", replay, replayed, err)
	}
	if replay.ID != first.ID || replay.Quantity != first.Quantity || !replay.UpdatedAt.Equal(first.UpdatedAt) {
		t.Fatalf("replay = %#v, want original result %#v", replay, first)
	}
	if _, _, err := service.Add(t.Context(), fixture.sandboxID, key.String(), consumptionmodels.AddBalanceRequest{
		CustomerID: fixture.customerID, MeterKey: fixture.meterKey, Quantity: 11,
	}); !errors.Is(err, consumptionmodels.ErrIdempotencyKeyConflict) {
		t.Fatalf("reused key error = %v, want conflict", err)
	}
	expiresAt := time.Now().Add(24 * time.Hour)
	if _, _, err := service.Add(t.Context(), fixture.sandboxID, key.String(), consumptionmodels.AddBalanceRequest{
		CustomerID: fixture.customerID, MeterKey: fixture.meterKey, Quantity: 10, ExpiresAt: &expiresAt,
	}); !errors.Is(err, consumptionmodels.ErrIdempotencyKeyConflict) {
		t.Fatalf("reused key with changed expiration error = %v, want conflict", err)
	}

	const sameKeyRequests = 8
	sameKey := newV7(t)
	sameKeyErrors := make(chan error, sameKeyRequests)
	resultKinds := make(chan bool, sameKeyRequests)
	var retries sync.WaitGroup
	for range sameKeyRequests {
		retries.Add(1)
		go func() {
			defer retries.Done()
			_, wasReplayed, addErr := service.Add(ctx, fixture.sandboxID, sameKey.String(), consumptionmodels.AddBalanceRequest{
				CustomerID: fixture.customerID, MeterKey: fixture.meterKey, Quantity: 2,
			})
			sameKeyErrors <- addErr
			resultKinds <- wasReplayed
		}()
	}
	retries.Wait()
	close(sameKeyErrors)
	close(resultKinds)
	for addErr := range sameKeyErrors {
		if addErr != nil {
			t.Fatalf("concurrent same-key addition: %v", addErr)
		}
	}
	replayCount := 0
	for wasReplayed := range resultKinds {
		if wasReplayed {
			replayCount++
		}
	}
	if replayCount != sameKeyRequests-1 {
		t.Fatalf("same-key replay count = %d, want %d", replayCount, sameKeyRequests-1)
	}

	const concurrentAdditions = 12
	errorsByAddition := make(chan error, concurrentAdditions)
	var additions sync.WaitGroup
	distinctKeys := make([]uuid.UUID, concurrentAdditions)
	for index := range concurrentAdditions {
		distinctKeys[index] = newV7(t)
	}
	for index := range concurrentAdditions {
		additions.Add(1)
		go func(additionKey uuid.UUID) {
			defer additions.Done()
			_, _, addErr := service.Add(ctx, fixture.sandboxID, additionKey.String(), consumptionmodels.AddBalanceRequest{
				CustomerID: fixture.customerID, MeterKey: fixture.meterKey, Quantity: 1,
			})
			errorsByAddition <- addErr
		}(distinctKeys[index])
	}
	additions.Wait()
	close(errorsByAddition)
	for addErr := range errorsByAddition {
		if addErr != nil {
			t.Fatalf("concurrent addition: %v", addErr)
		}
	}
	loaded, err := service.Get(t.Context(), fixture.sandboxID, fixture.customerID, fixture.meterKey)
	if err != nil {
		t.Fatalf("get concurrent result: %v", err)
	}
	if loaded.Quantity != 12+concurrentAdditions {
		t.Fatalf("quantity = %d, want %d", loaded.Quantity, 12+concurrentAdditions)
	}

	set, err := service.Set(t.Context(), fixture.sandboxID, fixture.customerID, fixture.meterKey, consumptionmodels.SetBalanceRequest{Quantity: 3})
	if err != nil || set.Quantity != 3 {
		t.Fatalf("set balance = %#v, error %v", set, err)
	}
	setAgain, err := service.Set(t.Context(), fixture.sandboxID, fixture.customerID, fixture.meterKey, consumptionmodels.SetBalanceRequest{Quantity: 3})
	if err != nil || setAgain.ID != set.ID || setAgain.Quantity != 3 {
		t.Fatalf("repeat exact set = %#v, error %v", setAgain, err)
	}

	live, _, err := service.Add(t.Context(), fixture.liveID, newV7(t).String(), consumptionmodels.AddBalanceRequest{
		CustomerID: fixture.customerID, MeterKey: fixture.meterKey, Quantity: 7,
	})
	if err != nil || live.Quantity != 7 {
		t.Fatalf("Live addition = %#v, error %v", live, err)
	}
	sandbox, err := service.Get(t.Context(), fixture.sandboxID, fixture.customerID, fixture.meterKey)
	if err != nil || sandbox.Quantity != 3 {
		t.Fatalf("Sandbox balance after Live addition = %#v, error %v", sandbox, err)
	}
}

func TestBalancesRequireExistingCustomersAndActiveMeters(t *testing.T) {
	pool := testdb.OpenMigratedDatabase(t)
	fixture := createBalanceFixture(t, pool)
	service := consumptionservices.NewBalanceService(consumptionrepositories.NewBalanceRepository(pool))

	if _, err := service.Get(t.Context(), fixture.sandboxID, fixture.customerID, fixture.meterKey); !errors.Is(err, consumptionmodels.ErrBalanceNotFound) {
		t.Fatalf("missing balance error = %v, want balance not found", err)
	}
	if _, err := service.List(t.Context(), fixture.sandboxID, "missing_customer"); !errors.Is(err, consumptionmodels.ErrBalanceSubjectNotFound) {
		t.Fatalf("missing customer list error = %v, want subject not found", err)
	}
	key := newV7(t)
	archived, _, err := service.Add(t.Context(), fixture.sandboxID, key.String(), consumptionmodels.AddBalanceRequest{
		CustomerID: fixture.customerID, MeterKey: fixture.archivableMeterKey, Quantity: 4,
	})
	if err != nil {
		t.Fatalf("create balance for archivable meter: %v", err)
	}
	if _, err := pool.Exec(t.Context(), `
		UPDATE project_environment_meters SET archived_at = now()
		WHERE project_environment_id = $1 AND meter_id = $2
	`, fixture.sandboxID, fixture.archivableMeterID); err != nil {
		t.Fatalf("archive meter: %v", err)
	}
	if _, err := service.Get(t.Context(), fixture.sandboxID, fixture.customerID, fixture.archivableMeterKey); !errors.Is(err, consumptionmodels.ErrBalanceSubjectNotFound) {
		t.Fatalf("archived meter get error = %v, want subject not found", err)
	}
	balances, err := service.List(t.Context(), fixture.sandboxID, fixture.customerID)
	if err != nil {
		t.Fatalf("list balances after archive: %v", err)
	}
	if len(balances) != 0 {
		t.Fatalf("active balances = %#v, want archived meter excluded", balances)
	}
	replay, replayed, err := service.Add(t.Context(), fixture.sandboxID, key.String(), consumptionmodels.AddBalanceRequest{
		CustomerID: fixture.customerID, MeterKey: fixture.archivableMeterKey, Quantity: 4,
	})
	if err != nil || !replayed || replay.ID != archived.ID {
		t.Fatalf("archived-meter idempotency replay = %#v, replayed %t, error %v", replay, replayed, err)
	}
}

type balanceFixture struct {
	sandboxID          uuid.UUID
	liveID             uuid.UUID
	customerID         string
	meterKey           string
	archivableMeterKey string
	archivableMeterID  uuid.UUID
}

func createBalanceFixture(t *testing.T, pool *pgxpool.Pool) balanceFixture {
	t.Helper()
	user, err := userservices.NewUserService(userrepositories.NewUserRepository(pool)).ResolveUser(t.Context(), "balance_test_owner")
	if err != nil {
		t.Fatalf("create owner: %v", err)
	}
	organization, _, err := organizationservices.NewOrganizationManagementService(
		organizationrepositories.NewOrganizationRepository(pool),
	).CreateOrganization(t.Context(), "Balance Test Organization", user.ID)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := projectservices.NewProjectManagementService(
		projectrepositories.NewProjectRepository(pool),
	).CreateProject(t.Context(), organization.ID, "Balance Test Project")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	sandboxID := balanceEnvironmentID(t, project, projectmodels.ProjectEnvironmentSandbox)
	liveID := balanceEnvironmentID(t, project, projectmodels.ProjectEnvironmentLive)
	customerService := customerservices.NewCustomerService(customerrepositories.NewCustomerRepository(pool))
	for _, environmentID := range []uuid.UUID{sandboxID, liveID} {
		if _, err := customerService.Create(t.Context(), environmentID, customermodels.CreateCustomerRequest{CustomerID: "customer_123"}); err != nil {
			t.Fatalf("create customer in %s: %v", environmentID, err)
		}
	}
	meterService := meterservices.NewMeterService(meterrepositories.NewMeterRepository(pool))
	var archivableMeterID uuid.UUID
	for _, environmentID := range []uuid.UUID{sandboxID, liveID} {
		if _, err := meterService.Create(t.Context(), environmentID, metermodels.CreateMeterRequest{
			MeterKey: "api_calls", Name: "API calls", Type: metermodels.MeterTypePrepaid,
		}); err != nil {
			t.Fatalf("create meter in %s: %v", environmentID, err)
		}
		meter, err := meterService.Create(t.Context(), environmentID, metermodels.CreateMeterRequest{
			MeterKey: "storage_gb", Name: "Storage", Type: metermodels.MeterTypePrepaid,
		})
		if err != nil {
			t.Fatalf("create archivable meter in %s: %v", environmentID, err)
		}
		archivableMeterID = meter.ID
	}
	return balanceFixture{
		sandboxID: sandboxID, liveID: liveID, customerID: "customer_123", meterKey: "api_calls",
		archivableMeterKey: "storage_gb", archivableMeterID: archivableMeterID,
	}
}

func balanceEnvironmentID(t *testing.T, project projectmodels.Project, name projectmodels.ProjectEnvironmentName) uuid.UUID {
	t.Helper()
	for _, environment := range project.Environments {
		if environment.Name == name {
			return environment.ID
		}
	}
	t.Fatalf("%s environment not found", name)
	return uuid.Nil
}

func newV7(t *testing.T) uuid.UUID {
	t.Helper()
	value, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("generate UUID v7: %v", err)
	}
	return value
}
