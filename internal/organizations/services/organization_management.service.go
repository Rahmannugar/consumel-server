package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Rahmannugar/consumel-server/internal/common/ids"
	"github.com/Rahmannugar/consumel-server/internal/organizations/models"
	"github.com/google/uuid"
)

var (
	ErrOrganizationNameRequired = errors.New("organization name is required")
	ErrOwnerUserIDRequired      = errors.New("owner user ID is required")
)

type OrganizationRepository interface {
	CreateOrganizationWithOwner(
		context.Context,
		models.Organization,
		[]models.OrganizationRole,
		models.OrganizationMembership,
	) (models.Organization, models.OrganizationMembership, error)
}

type OrganizationManagementService struct {
	repository OrganizationRepository
}

func NewOrganizationManagementService(
	repository OrganizationRepository,
) *OrganizationManagementService {
	return &OrganizationManagementService{repository: repository}
}

func (service *OrganizationManagementService) CreateOrganization(
	ctx context.Context,
	name string,
	ownerUserID uuid.UUID,
) (models.Organization, models.OrganizationMembership, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return models.Organization{}, models.OrganizationMembership{}, ErrOrganizationNameRequired
	}
	if ownerUserID == uuid.Nil {
		return models.Organization{}, models.OrganizationMembership{}, ErrOwnerUserIDRequired
	}

	organizationID, err := ids.New()
	if err != nil {
		return models.Organization{}, models.OrganizationMembership{}, fmt.Errorf("generate organization ID: %w", err)
	}

	organization := models.Organization{
		ID:          organizationID,
		OwnerUserID: ownerUserID,
		Name:        name,
	}
	adminRoleID, err := ids.New()
	if err != nil {
		return models.Organization{}, models.OrganizationMembership{}, fmt.Errorf("generate Admin role ID: %w", err)
	}
	developerRoleID, err := ids.New()
	if err != nil {
		return models.Organization{}, models.OrganizationMembership{}, fmt.Errorf("generate Developer role ID: %w", err)
	}
	adminSystemKey := models.OrganizationRoleSystemKeyAdmin
	developerSystemKey := models.OrganizationRoleSystemKeyDeveloper
	roles := []models.OrganizationRole{
		{
			ID:             adminRoleID,
			OrganizationID: organizationID,
			Name:           "Admin",
			SystemKey:      &adminSystemKey,
		},
		{
			ID:             developerRoleID,
			OrganizationID: organizationID,
			Name:           "Developer",
			SystemKey:      &developerSystemKey,
		},
	}
	membership := models.OrganizationMembership{
		OrganizationID: organizationID,
		UserID:         ownerUserID,
		RoleID:         adminRoleID,
		Status:         models.OrganizationMembershipStatusActive,
	}

	return service.repository.CreateOrganizationWithOwner(ctx, organization, roles, membership)
}
