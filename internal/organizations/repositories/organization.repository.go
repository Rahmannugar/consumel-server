package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/Rahmannugar/consumel-server/internal/organizations/models"
	organizationdb "github.com/Rahmannugar/consumel-server/internal/organizations/repositories/generated"
	"github.com/jackc/pgx/v5/pgtype"
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
	roles []models.OrganizationRole,
	membership models.OrganizationMembership,
) (models.Organization, models.OrganizationMembership, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return models.Organization{}, models.OrganizationMembership{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	queries := repository.queries.WithTx(tx)
	createdOrganization, err := queries.CreateOrganization(ctx, organizationdb.CreateOrganizationParams{
		ID:          organization.ID,
		OwnerUserID: organization.OwnerUserID,
		Name:        organization.Name,
	})
	if err != nil {
		return models.Organization{}, models.OrganizationMembership{}, fmt.Errorf("create organization: %w", err)
	}

	for _, role := range roles {
		_, err := queries.CreateOrganizationRole(ctx, organizationdb.CreateOrganizationRoleParams{
			ID:             role.ID,
			OrganizationID: role.OrganizationID,
			Name:           role.Name,
			SystemKey:      nullableRoleSystemKey(role.SystemKey),
		})
		if err != nil {
			return models.Organization{}, models.OrganizationMembership{}, fmt.Errorf("create organization role: %w", err)
		}
	}

	createdMembership, err := queries.CreateOrganizationMembership(
		ctx,
		organizationdb.CreateOrganizationMembershipParams{
			OrganizationID: membership.OrganizationID,
			UserID:         membership.UserID,
			RoleID:         membership.RoleID,
			Status:         string(membership.Status),
		},
	)
	if err != nil {
		return models.Organization{}, models.OrganizationMembership{}, fmt.Errorf("create owner membership: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Organization{}, models.OrganizationMembership{}, fmt.Errorf("commit transaction: %w", err)
	}

	return mapOrganization(createdOrganization), mapCreatedOrganizationMembership(createdMembership), nil
}

func mapCreatedOrganizationMembership(
	membership organizationdb.CreateOrganizationMembershipRow,
) models.OrganizationMembership {
	return models.OrganizationMembership{
		OrganizationID: membership.OrganizationID,
		UserID:         membership.UserID,
		RoleID:         membership.RoleID,
		Status:         models.OrganizationMembershipStatus(membership.Status),
		CreatedAt:      membership.CreatedAt.Time,
		UpdatedAt:      membership.UpdatedAt.Time,
		RemovedAt:      nullableTime(membership.RemovedAt),
	}
}

func mapOrganization(organization organizationdb.CreateOrganizationRow) models.Organization {
	return models.Organization{
		ID:          organization.ID,
		OwnerUserID: organization.OwnerUserID,
		Name:        organization.Name,
		CreatedAt:   organization.CreatedAt.Time,
		UpdatedAt:   organization.UpdatedAt.Time,
		DeletedAt:   nullableTime(organization.DeletedAt),
		SuspendedAt: nullableTime(organization.SuspendedAt),
	}
}

func nullableTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	timestamp := value.Time
	return &timestamp
}
