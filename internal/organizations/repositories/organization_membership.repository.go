package repositories

import (
	"context"
	"fmt"

	"github.com/Rahmannugar/consumel-server/internal/organizations/models"
	organizationdb "github.com/Rahmannugar/consumel-server/internal/organizations/repositories/generated"
	"github.com/google/uuid"
)

func (repository *OrganizationRepository) OrganizationMembership(
	ctx context.Context,
	organizationID uuid.UUID,
	userID uuid.UUID,
) (models.OrganizationMembership, error) {
	membership, err := repository.queries.GetOrganizationMembership(
		ctx,
		organizationdb.GetOrganizationMembershipParams{
			OrganizationID: organizationID,
			UserID:         userID,
		},
	)
	if err != nil {
		return models.OrganizationMembership{}, fmt.Errorf("get organization membership: %w", err)
	}

	return mapOrganizationMembership(membership), nil
}

func mapOrganizationMembership(
	membership organizationdb.GetOrganizationMembershipRow,
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
