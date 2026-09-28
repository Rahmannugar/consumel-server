package unit_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"

	authenticationmodels "github.com/Rahmannugar/consumel-server/internal/authentication/models"
	authenticationservices "github.com/Rahmannugar/consumel-server/internal/authentication/services"
)

type apiKeyRepository struct {
	wantHash [sha256.Size]byte
	found    bool
}

func (repository apiKeyRepository) ResolveActiveProjectAPIKey(
	_ context.Context,
	hash []byte,
) (authenticationmodels.APIKeyContext, bool, error) {
	if string(hash) != string(repository.wantHash[:]) {
		return authenticationmodels.APIKeyContext{}, false, errors.New("unexpected API key hash")
	}
	return authenticationmodels.APIKeyContext{Environment: "sandbox"}, repository.found, nil
}

func TestAPIKeyAuthenticatorAcceptsOnlyResolvedConsumelBearerKeys(t *testing.T) {
	secret := "cm_test_3xKq7VfJm2zY8wN4aBcD6eFgH9iLpQrStUvWx0Z1A2B"
	hash := sha256.Sum256([]byte(secret))
	authenticator := authenticationservices.NewAPIKeyAuthenticator(apiKeyRepository{
		wantHash: hash, found: true,
	})

	resolved, err := authenticator.Authenticate(t.Context(), "Bearer "+secret)
	if err != nil {
		t.Fatalf("authenticate active key: %v", err)
	}
	if resolved.Environment != "sandbox" {
		t.Fatalf("environment = %q, want sandbox", resolved.Environment)
	}
}

func TestAPIKeyAuthenticatorRejectsMalformedAndUnknownKeys(t *testing.T) {
	secret := "cm_test_3xKq7VfJm2zY8wN4aBcD6eFgH9iLpQrStUvWx0Z1A2B"
	hash := sha256.Sum256([]byte(secret))
	cases := []struct {
		name          string
		authorization string
		found         bool
	}{
		{name: "missing", authorization: ""},
		{name: "wrong scheme", authorization: "Basic " + secret},
		{name: "wrong prefix", authorization: "Bearer invalid_3xKq7VfJm2zY8wN4aBcD6eFgH9iLpQrStUvWx0Z1A2B"},
		{name: "unknown", authorization: "Bearer " + secret, found: false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			authenticator := authenticationservices.NewAPIKeyAuthenticator(apiKeyRepository{
				wantHash: hash, found: test.found,
			})
			_, err := authenticator.Authenticate(t.Context(), test.authorization)
			if !errors.Is(err, authenticationservices.ErrInvalidAPIKey) {
				t.Fatalf("error = %v, want invalid API key", err)
			}
		})
	}
}
