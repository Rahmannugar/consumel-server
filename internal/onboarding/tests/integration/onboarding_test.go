//go:build integration

package integration_test

import (
	"bytes"
	"crypto/rand"
	"sync"
	"testing"
	"time"

	authlierpostgres "github.com/Rahmannugar/authlier/storage/postgres"
	"github.com/Rahmannugar/consumel-server/internal/infra/database/testdb"
	"github.com/Rahmannugar/consumel-server/internal/infra/emaildelivery"
	onboardingmodels "github.com/Rahmannugar/consumel-server/internal/onboarding/models"
	onboardingrepositories "github.com/Rahmannugar/consumel-server/internal/onboarding/repositories"
	onboardingservices "github.com/Rahmannugar/consumel-server/internal/onboarding/services"
	userrepositories "github.com/Rahmannugar/consumel-server/internal/users/repositories"
	userservices "github.com/Rahmannugar/consumel-server/internal/users/services"
)

func TestConcurrentOnboardingCreatesOneCompleteSetupAndOneWelcomeEmail(t *testing.T) {
	pool := testdb.OpenMigratedDatabase(t)
	authlierDatabase, err := authlierpostgres.New(pool, authlierpostgres.Config{})
	if err != nil {
		t.Fatalf("create Authlier database: %v", err)
	}
	if err := authlierDatabase.Migrate(t.Context()); err != nil {
		t.Fatalf("migrate Authlier database: %v", err)
	}

	subjectID := "onboarding-subject"
	webauthnHandle := make([]byte, 64)
	if _, err := rand.Read(webauthnHandle); err != nil {
		t.Fatalf("generate WebAuthn handle: %v", err)
	}
	if _, err := pool.Exec(t.Context(), `INSERT INTO authlier_users
		(id, email, email_verified, webauthn_handle, created_at)
		VALUES ($1, $2, true, $3, $4)`,
		subjectID, "builder@example.com", webauthnHandle, time.Now().UTC(),
	); err != nil {
		t.Fatalf("create Authlier user: %v", err)
	}

	userService := userservices.NewUserService(userrepositories.NewUserRepository(pool))
	user, err := userService.ResolveUser(t.Context(), subjectID)
	if err != nil {
		t.Fatalf("resolve tenant user: %v", err)
	}
	queue, err := emaildelivery.NewQueue(pool, bytes.Repeat([]byte{0x4f}, 32))
	if err != nil {
		t.Fatalf("create email queue: %v", err)
	}
	service := onboardingservices.New(
		onboardingrepositories.New(pool, queue),
		"https://app.consumel.test",
	)

	request := onboardingmodels.SetupRequest{
		OrganizationName: "Acme, Inc.",
		ProjectName:      "Acme API",
	}
	results := make([]onboardingmodels.Setup, 2)
	errors := make([]error, 2)
	var waitGroup sync.WaitGroup
	start := make(chan struct{})
	for index := range results {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			<-start
			results[index], errors[index] = service.Setup(t.Context(), subjectID, user.ID, request)
		}()
	}
	close(start)
	waitGroup.Wait()

	for _, setupError := range errors {
		if setupError != nil {
			t.Fatalf("set up first project concurrently: %v", setupError)
		}
	}
	if results[0].Project.ID != results[1].Project.ID {
		t.Fatalf("concurrent project IDs differ: %s and %s", results[0].Project.ID, results[1].Project.ID)
	}
	if results[0].Project.Slug != "acme-api" || results[1].Project.Slug != "acme-api" {
		t.Fatalf(
			"concurrent project slugs = %q and %q, want acme-api",
			results[0].Project.Slug,
			results[1].Project.Slug,
		)
	}
	if results[0].Created == results[1].Created {
		t.Fatalf("created outcomes = %t and %t, want exactly one true", results[0].Created, results[1].Created)
	}

	var organizations, projects, environments, deliveries, outboxEvents int
	if err := pool.QueryRow(t.Context(), `SELECT
		(SELECT count(*) FROM organizations),
		(SELECT count(*) FROM projects),
		(SELECT count(*) FROM project_environments),
		(SELECT count(*) FROM email_deliveries WHERE template = 'welcome'),
		(SELECT count(*) FROM outbox_events WHERE event_type = 'email.delivery.queued.v1')`,
	).Scan(&organizations, &projects, &environments, &deliveries, &outboxEvents); err != nil {
		t.Fatalf("inspect onboarding transaction: %v", err)
	}
	if organizations != 1 || projects != 1 || environments != 2 {
		t.Fatalf(
			"persisted setup = organizations:%d projects:%d environments:%d, want 1, 1, 2",
			organizations, projects, environments,
		)
	}
	if deliveries != 1 || outboxEvents != 1 {
		t.Fatalf("welcome work = deliveries:%d outbox:%d, want 1, 1", deliveries, outboxEvents)
	}
}
