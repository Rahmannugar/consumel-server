//go:build integration

package integration_test

import (
	"sync"
	"testing"

	"github.com/Rahmannugar/consumel-server/internal/infra/database/testdb"
	"github.com/Rahmannugar/consumel-server/internal/users/repositories"
	"github.com/Rahmannugar/consumel-server/internal/users/services"
	"github.com/google/uuid"
)

func TestConcurrentFirstUseResolvesOneConsumelUser(t *testing.T) {
	pool := testdb.OpenMigratedDatabase(t)
	service := services.NewUserService(repositories.NewUserRepository(pool))

	const callers = 16
	start := make(chan struct{})
	results := make(chan uuid.UUID, callers)
	errors := make(chan error, callers)
	var group sync.WaitGroup
	for range callers {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			user, err := service.ResolveUser(t.Context(), "authlier-subject-concurrent")
			if err != nil {
				errors <- err
				return
			}
			results <- user.ID
		}()
	}
	close(start)
	group.Wait()
	close(results)
	close(errors)

	for err := range errors {
		t.Fatalf("resolve user concurrently: %v", err)
	}
	var resolvedID uuid.UUID
	for id := range results {
		if resolvedID == uuid.Nil {
			resolvedID = id
		}
		if id != resolvedID {
			t.Fatalf("resolved IDs differ: got %s and %s", resolvedID, id)
		}
	}
	if resolvedID == uuid.Nil {
		t.Fatal("no user was resolved")
	}
}
