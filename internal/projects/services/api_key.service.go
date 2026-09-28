package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/Rahmannugar/consumel-server/internal/common/ids"
	"github.com/Rahmannugar/consumel-server/internal/projects/models"
	"github.com/google/uuid"
)

var ErrEnvironmentInvalid = errors.New("environment must be sandbox or live")

func (service *ProjectManagementService) ActiveAPIKey(
	ctx context.Context,
	userID uuid.UUID,
	projectID uuid.UUID,
	environment models.ProjectEnvironmentName,
) (*models.APIKey, error) {
	if err := validateAPIKeyContext(userID, projectID, environment); err != nil {
		return nil, err
	}
	return service.repository.ActiveAPIKey(ctx, userID, projectID, environment)
}

func (service *ProjectManagementService) ActivateEnvironment(
	ctx context.Context,
	userID uuid.UUID,
	projectID uuid.UUID,
	environment models.ProjectEnvironmentName,
) (models.ProjectEnvironment, error) {
	if err := validateAPIKeyContext(userID, projectID, environment); err != nil {
		return models.ProjectEnvironment{}, err
	}
	return service.repository.ActivateEnvironment(ctx, userID, projectID, environment)
}

func (service *ProjectManagementService) CreateAPIKey(
	ctx context.Context,
	userID uuid.UUID,
	projectID uuid.UUID,
	environment models.ProjectEnvironmentName,
) (models.CreatedAPIKey, error) {
	if err := validateAPIKeyContext(userID, projectID, environment); err != nil {
		return models.CreatedAPIKey{}, err
	}
	key, hash, secret, err := newAPIKey(environment)
	if err != nil {
		return models.CreatedAPIKey{}, err
	}
	created, err := service.repository.CreateAPIKey(
		ctx, userID, projectID, environment, key, hash[:],
	)
	if err != nil {
		return models.CreatedAPIKey{}, err
	}
	return models.CreatedAPIKey{APIKey: created, Secret: secret}, nil
}

func (service *ProjectManagementService) ReplaceAPIKey(
	ctx context.Context,
	userID uuid.UUID,
	projectID uuid.UUID,
	environment models.ProjectEnvironmentName,
) (models.CreatedAPIKey, error) {
	if err := validateAPIKeyContext(userID, projectID, environment); err != nil {
		return models.CreatedAPIKey{}, err
	}
	key, hash, secret, err := newAPIKey(environment)
	if err != nil {
		return models.CreatedAPIKey{}, err
	}
	created, err := service.repository.ReplaceAPIKey(
		ctx, userID, projectID, environment, key, hash[:],
	)
	if err != nil {
		return models.CreatedAPIKey{}, err
	}
	return models.CreatedAPIKey{APIKey: created, Secret: secret}, nil
}

func (service *ProjectManagementService) RevokeAPIKey(
	ctx context.Context,
	userID uuid.UUID,
	projectID uuid.UUID,
	environment models.ProjectEnvironmentName,
) error {
	if err := validateAPIKeyContext(userID, projectID, environment); err != nil {
		return err
	}
	return service.repository.RevokeAPIKey(ctx, userID, projectID, environment)
}

func validateAPIKeyContext(
	userID uuid.UUID,
	projectID uuid.UUID,
	environment models.ProjectEnvironmentName,
) error {
	if userID == uuid.Nil {
		return ErrUserIDRequired
	}
	if projectID == uuid.Nil {
		return models.ErrProjectEnvironmentUnavailable
	}
	if environment != models.ProjectEnvironmentSandbox &&
		environment != models.ProjectEnvironmentLive {
		return ErrEnvironmentInvalid
	}
	return nil
}

func newAPIKey(
	environment models.ProjectEnvironmentName,
) (models.APIKey, [sha256.Size]byte, string, error) {
	id, err := ids.New()
	if err != nil {
		return models.APIKey{}, [sha256.Size]byte{}, "", fmt.Errorf("generate API key ID: %w", err)
	}
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return models.APIKey{}, [sha256.Size]byte{}, "", fmt.Errorf("generate API key secret: %w", err)
	}
	prefix := "cm_test_"
	if environment == models.ProjectEnvironmentLive {
		prefix = "cm_live_"
	}
	secret := prefix + base64.RawURLEncoding.EncodeToString(random)
	hash := sha256.Sum256([]byte(secret))
	return models.APIKey{
		ID: id, Environment: environment, Prefix: prefix,
		LastFour: secret[len(secret)-4:],
	}, hash, secret, nil
}
