package repositories

import (
	"context"
	"errors"
	"fmt"

	authenticationmodels "github.com/Rahmannugar/consumel-server/internal/authentication/models"
	authenticationdb "github.com/Rahmannugar/consumel-server/internal/authentication/repositories/generated"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type APIKeyContextRepository struct {
	queries *authenticationdb.Queries
}

func NewAPIKeyContextRepository(pool *pgxpool.Pool) *APIKeyContextRepository {
	return &APIKeyContextRepository{queries: authenticationdb.New(pool)}
}

func (repository *APIKeyContextRepository) ResolveActiveProjectAPIKey(
	ctx context.Context,
	hash []byte,
) (authenticationmodels.APIKeyContext, bool, error) {
	row, err := repository.queries.ResolveActiveProjectAPIKey(ctx, hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return authenticationmodels.APIKeyContext{}, false, nil
	}
	if err != nil {
		return authenticationmodels.APIKeyContext{}, false, fmt.Errorf("resolve active project API key: %w", err)
	}
	return authenticationmodels.APIKeyContext{
		APIKeyID: row.ApiKeyID, OrganizationID: row.OrganizationID,
		ProjectID: row.ProjectID, ProjectEnvironmentID: row.ProjectEnvironmentID,
		Environment: row.Environment,
	}, true, nil
}
