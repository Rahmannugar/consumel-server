//go:build integration

package integration_test

import (
	"testing"

	"github.com/Rahmannugar/consumel-server/internal/infra/database/testdb"
	"github.com/Rahmannugar/consumel-server/internal/users/repositories"
	"github.com/Rahmannugar/consumel-server/internal/users/services"
)

func TestUserPersistence(t *testing.T) {
	pool := testdb.OpenMigratedDatabase(t)
	repository := repositories.NewUserRepository(pool)
	service := services.NewUserService(repository)

	user, err := service.CreateUser(t.Context(), "user_clerk_123")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if user.ID.Version() != 7 {
		t.Fatalf("user ID version = %d, want 7", user.ID.Version())
	}

	storedUser, err := repository.UserByClerkID(t.Context(), "user_clerk_123")
	if err != nil {
		t.Fatalf("get user by Clerk ID: %v", err)
	}
	if storedUser != user {
		t.Fatalf("stored user = %#v, want %#v", storedUser, user)
	}
}
