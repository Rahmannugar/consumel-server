package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/Rahmannugar/consumel-server/internal/users/models"
	userdb "github.com/Rahmannugar/consumel-server/internal/users/repositories/generated"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	queries *userdb.Queries
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{queries: userdb.New(pool)}
}

func (repository *UserRepository) ResolveUserByAuthlierSubjectID(
	ctx context.Context,
	user models.User,
) (models.User, error) {
	// Established users need one read and no write. Only first access reaches
	// the insert path below.
	existing, err := repository.queries.GetUserByAuthlierSubjectID(ctx, user.AuthlierSubjectID)
	if err == nil {
		return mapUser(existing), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return models.User{}, fmt.Errorf("find user by Authlier subject ID: %w", err)
	}

	// The unique subject constraint prevents concurrent first requests from
	// creating duplicates.
	resolved, err := repository.queries.ResolveUserByAuthlierSubjectID(
		ctx,
		userdb.ResolveUserByAuthlierSubjectIDParams{
			ID:                user.ID,
			AuthlierSubjectID: user.AuthlierSubjectID,
		},
	)
	// If another request inserts the user while this query is running, read the
	// row once more after that insert commits.
	if errors.Is(err, pgx.ErrNoRows) {
		existing, findErr := repository.queries.GetUserByAuthlierSubjectID(ctx, user.AuthlierSubjectID)
		if findErr != nil {
			return models.User{}, fmt.Errorf("resolve concurrent user by Authlier subject ID: %w", findErr)
		}
		return mapUser(existing), nil
	}
	if err != nil {
		return models.User{}, fmt.Errorf("resolve user by Authlier subject ID: %w", err)
	}

	return models.User{
		ID:                resolved.ID,
		AuthlierSubjectID: resolved.AuthlierSubjectID,
		CreatedAt:         resolved.CreatedAt.Time,
	}, nil
}

func (repository *UserRepository) UserByAuthlierSubjectID(
	ctx context.Context,
	authlierSubjectID string,
) (models.User, error) {
	user, err := repository.queries.GetUserByAuthlierSubjectID(ctx, authlierSubjectID)
	if err != nil {
		return models.User{}, fmt.Errorf("get user by Authlier subject ID: %w", err)
	}

	return mapUser(user), nil
}

func mapUser(user userdb.User) models.User {
	return models.User{
		ID:                user.ID,
		AuthlierSubjectID: user.AuthlierSubjectID,
		CreatedAt:         user.CreatedAt.Time,
	}
}
