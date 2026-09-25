package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	authenticationmodels "github.com/Rahmannugar/consumel-server/internal/authentication/models"
	organizationmodels "github.com/Rahmannugar/consumel-server/internal/organizations/models"
	usermodels "github.com/Rahmannugar/consumel-server/internal/users/models"
)

var ErrUnauthenticated = errors.New("unauthenticated")

type SessionResolver interface {
	ResolveSession(*http.Request) (authenticationmodels.Session, error)
}

type UserResolver interface {
	ResolveUser(context.Context, string) (usermodels.User, error)
}

type AccountContextRepository interface {
	AccountContextByAuthlierSubjectID(
		context.Context,
		string,
	) (usermodels.User, []organizationmodels.OrganizationAccess, bool, error)
}

type AuthenticatedTenantService struct {
	sessions SessionResolver
	users    UserResolver
	accounts AccountContextRepository
}

func NewAuthenticatedTenantService(
	sessions SessionResolver,
	users UserResolver,
	accounts AccountContextRepository,
) *AuthenticatedTenantService {
	return &AuthenticatedTenantService{
		sessions: sessions,
		users:    users,
		accounts: accounts,
	}
}

func (service *AuthenticatedTenantService) Resolve(
	request *http.Request,
) (authenticationmodels.AuthenticatedTenant, error) {
	session, err := service.sessions.ResolveSession(request)
	if err != nil {
		return authenticationmodels.AuthenticatedTenant{}, fmt.Errorf("%w: %v", ErrUnauthenticated, err)
	}

	user, access, found, err := service.accounts.AccountContextByAuthlierSubjectID(
		request.Context(),
		session.SubjectID,
	)
	if err != nil {
		return authenticationmodels.AuthenticatedTenant{}, fmt.Errorf("resolve account context: %w", err)
	}
	if !found {
		// Authlier can authenticate a subject before Consumel has created its local user projection.
		if _, err := service.users.ResolveUser(request.Context(), session.SubjectID); err != nil {
			return authenticationmodels.AuthenticatedTenant{}, fmt.Errorf("create Consumel user: %w", err)
		}
		user, access, found, err = service.accounts.AccountContextByAuthlierSubjectID(
			request.Context(),
			session.SubjectID,
		)
		if err != nil {
			return authenticationmodels.AuthenticatedTenant{}, fmt.Errorf("resolve created account context: %w", err)
		}
		if !found {
			return authenticationmodels.AuthenticatedTenant{}, fmt.Errorf("created Consumel user is unavailable")
		}
	}

	return authenticationmodels.AuthenticatedTenant{
		Session:            session,
		User:               user,
		OrganizationAccess: access,
	}, nil
}
