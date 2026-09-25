package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	authenticationmodels "github.com/Rahmannugar/consumel-server/internal/authentication/models"
	organizationmodels "github.com/Rahmannugar/consumel-server/internal/organizations/models"
	usermodels "github.com/Rahmannugar/consumel-server/internal/users/models"
	"github.com/google/uuid"
)

var ErrUnauthenticated = errors.New("unauthenticated")

type SessionResolver interface {
	ResolveSession(*http.Request) (authenticationmodels.Session, error)
}

type UserResolver interface {
	ResolveUser(context.Context, string) (usermodels.User, error)
}

type OrganizationAccessRepository interface {
	ActiveOrganizationAccessByUser(context.Context, uuid.UUID) ([]organizationmodels.OrganizationAccess, error)
}

type AuthenticatedTenantService struct {
	sessions      SessionResolver
	users         UserResolver
	organizations OrganizationAccessRepository
}

func NewAuthenticatedTenantService(
	sessions SessionResolver,
	users UserResolver,
	organizations OrganizationAccessRepository,
) *AuthenticatedTenantService {
	return &AuthenticatedTenantService{
		sessions:      sessions,
		users:         users,
		organizations: organizations,
	}
}

func (service *AuthenticatedTenantService) Resolve(
	request *http.Request,
) (authenticationmodels.AuthenticatedTenant, error) {
	session, err := service.sessions.ResolveSession(request)
	if err != nil {
		return authenticationmodels.AuthenticatedTenant{}, fmt.Errorf("%w: %v", ErrUnauthenticated, err)
	}

	user, err := service.users.ResolveUser(request.Context(), session.SubjectID)
	if err != nil {
		return authenticationmodels.AuthenticatedTenant{}, fmt.Errorf("resolve Consumel user: %w", err)
	}

	access, err := service.organizations.ActiveOrganizationAccessByUser(request.Context(), user.ID)
	if err != nil {
		return authenticationmodels.AuthenticatedTenant{}, fmt.Errorf("resolve organization access: %w", err)
	}

	return authenticationmodels.AuthenticatedTenant{
		Session:            session,
		User:               user,
		OrganizationAccess: access,
	}, nil
}
