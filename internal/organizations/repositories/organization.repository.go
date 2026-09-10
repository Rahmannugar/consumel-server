package repositories

import (
	"context"
	"fmt"

	"github.com/Rahmannugar/consumel-server/internal/organizations/models"
	organizationdb "github.com/Rahmannugar/consumel-server/internal/organizations/repositories/generated"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrganizationRepository struct {
	pool    *pgxpool.Pool
	queries *organizationdb.Queries
}

func NewOrganizationRepository(pool *pgxpool.Pool) *OrganizationRepository {
	return &OrganizationRepository{
		pool:    pool,
		queries: organizationdb.New(pool),
	}
}

func (repository *OrganizationRepository) CreateOrganizationWithOwner(
	ctx context.Context,
	organization models.Organization,
	membership models.OrganizationMembership,
) (models.Organization, models.OrganizationMembership, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return models.Organization{}, models.OrganizationMembership{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	queries := repository.queries.WithTx(tx)
	createdOrganization, err := queries.CreateOrganization(ctx, organizationdb.CreateOrganizationParams{
		ID:   organization.ID,
		Name: organization.Name,
	})
	if err != nil {
		return models.Organization{}, models.OrganizationMembership{}, fmt.Errorf("create organization: %w", err)
	}

	createdMembership, err := queries.CreateOrganizationMembership(
		ctx,
		organizationdb.CreateOrganizationMembershipParams{
			OrganizationID: membership.OrganizationID,
			UserID:         membership.UserID,
			Role:           string(membership.Role),
			Status:         string(membership.Status),
		},
	)
	if err != nil {
		return models.Organization{}, models.OrganizationMembership{}, fmt.Errorf("create owner membership: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Organization{}, models.OrganizationMembership{}, fmt.Errorf("commit transaction: %w", err)
	}

	return mapOrganization(createdOrganization), mapOrganizationMembership(createdMembership), nil
}

func mapOrganization(organization organizationdb.Organization) models.Organization {
	return models.Organization{
		ID:        organization.ID,
		Name:      organization.Name,
		CreatedAt: organization.CreatedAt.Time,
		UpdatedAt: organization.UpdatedAt.Time,
	}
}
