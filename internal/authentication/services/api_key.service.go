package services

import (
	"context"
	"crypto/sha256"
	"errors"
	"strings"

	authenticationmodels "github.com/Rahmannugar/consumel-server/internal/authentication/models"
)

const projectAPIKeyLength = 51

var ErrInvalidAPIKey = errors.New("invalid project API key")

type APIKeyContextRepository interface {
	ResolveActiveProjectAPIKey(context.Context, []byte) (authenticationmodels.APIKeyContext, bool, error)
}

type APIKeyAuthenticator struct {
	repository APIKeyContextRepository
}

func NewAPIKeyAuthenticator(repository APIKeyContextRepository) *APIKeyAuthenticator {
	return &APIKeyAuthenticator{repository: repository}
}

func (authenticator *APIKeyAuthenticator) Authenticate(
	ctx context.Context,
	authorization string,
) (authenticationmodels.APIKeyContext, error) {
	secret, ok := bearerSecret(authorization)
	if !ok {
		return authenticationmodels.APIKeyContext{}, ErrInvalidAPIKey
	}
	hash := sha256.Sum256([]byte(secret))
	resolved, found, err := authenticator.repository.ResolveActiveProjectAPIKey(ctx, hash[:])
	if err != nil {
		return authenticationmodels.APIKeyContext{}, err
	}
	if !found {
		return authenticationmodels.APIKeyContext{}, ErrInvalidAPIKey
	}
	return resolved, nil
}

func bearerSecret(authorization string) (string, bool) {
	parts := strings.Fields(authorization)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	secret := parts[1]
	if len(secret) != projectAPIKeyLength ||
		(!strings.HasPrefix(secret, "cm_test_") && !strings.HasPrefix(secret, "cm_live_")) {
		return "", false
	}
	for _, value := range secret[8:] {
		if (value >= 'a' && value <= 'z') || (value >= 'A' && value <= 'Z') ||
			(value >= '0' && value <= '9') || value == '-' || value == '_' {
			continue
		}
		return "", false
	}
	return secret, true
}
