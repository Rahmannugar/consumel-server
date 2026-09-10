package repositories

import (
	"context"
	"fmt"

	"github.com/Rahmannugar/consumel-server/internal/organizations/models"
	organizationdb "github.com/Rahmannugar/consumel-server/internal/organizations/repositories/generated"
	"github.com/google/uuid"
)

func (repository *OrganizationRepository) OrganizationRoles(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]models.OrganizationRole, error) {
	roles, err := repository.queries.ListOrganizationRoles(ctx, organizationID)
	if err != nil {
		return nil, fmt.Errorf("list organization roles: %w", err)
	}

	mapped := make([]models.OrganizationRole, 0, len(roles))
	for _, role := range roles {
		mapped = append(mapped, mapOrganizationRole(role))
	}
	return mapped, nil
}

func mapOrganizationRole(role organizationdb.OrganizationRole) models.OrganizationRole {
	return models.OrganizationRole{
		ID:             role.ID,
		OrganizationID: role.OrganizationID,
		Name:           role.Name,
		SystemKey:      roleSystemKey(role.SystemKey),
		CreatedAt:      role.CreatedAt.Time,
		UpdatedAt:      role.UpdatedAt.Time,
		DeletedAt:      nullableTime(role.DeletedAt),
	}
}

func nullableRoleSystemKey(value *models.OrganizationRoleSystemKey) *string {
	if value == nil {
		return nil
	}
	key := string(*value)
	return &key
}

func roleSystemKey(value *string) *models.OrganizationRoleSystemKey {
	if value == nil {
		return nil
	}
	key := models.OrganizationRoleSystemKey(*value)
	return &key
}
