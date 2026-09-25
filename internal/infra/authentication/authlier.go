package authentication

import (
	"net/http"

	"github.com/Rahmannugar/authlier"
	authenticationmodels "github.com/Rahmannugar/consumel-server/internal/authentication/models"
)

type AuthlierSessionResolver struct {
	auth *authlier.Auth
}

func NewAuthlierSessionResolver(auth *authlier.Auth) *AuthlierSessionResolver {
	return &AuthlierSessionResolver{auth: auth}
}

func (resolver *AuthlierSessionResolver) ResolveSession(
	request *http.Request,
) (authenticationmodels.Session, error) {
	session, err := resolver.auth.ResolveSession(request)
	if err != nil {
		return authenticationmodels.Session{}, err
	}
	return authenticationmodels.Session{
		ID:        session.ID,
		SubjectID: session.SubjectID,
		CreatedAt: session.CreatedAt,
		ExpiresAt: session.ExpiresAt,
	}, nil
}
