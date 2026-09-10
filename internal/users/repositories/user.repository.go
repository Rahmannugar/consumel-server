package repositories

import (
	"context"
	"fmt"

	"github.com/Rahmannugar/consumel-server/internal/users/models"
	userdb "github.com/Rahmannugar/consumel-server/internal/users/repositories/generated"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	queries *userdb.Queries
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{queries: userdb.New(pool)}
}

func (repository *UserRepository) CreateUser(
	ctx context.Context,
	user models.User,
) (models.User, error) {
	created, err := repository.queries.CreateUser(ctx, userdb.CreateUserParams{
		ID:          user.ID,
		ClerkUserID: user.ClerkUserID,
	})
	if err != nil {
		return models.User{}, fmt.Errorf("create user: %w", err)
	}

	return mapUser(created), nil
}

func (repository *UserRepository) UserByClerkID(
	ctx context.Context,
	clerkUserID string,
) (models.User, error) {
	user, err := repository.queries.GetUserByClerkID(ctx, clerkUserID)
	if err != nil {
		return models.User{}, fmt.Errorf("get user by Clerk ID: %w", err)
	}

	return mapUser(user), nil
}

func mapUser(user userdb.User) models.User {
	return models.User{
		ID:          user.ID,
		ClerkUserID: user.ClerkUserID,
		CreatedAt:   user.CreatedAt.Time,
	}
}
