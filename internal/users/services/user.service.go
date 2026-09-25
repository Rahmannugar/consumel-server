package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Rahmannugar/consumel-server/internal/common/ids"
	"github.com/Rahmannugar/consumel-server/internal/users/models"
)

var ErrAuthlierSubjectIDRequired = errors.New("Authlier subject ID is required")

type UserRepository interface {
	ResolveUserByAuthlierSubjectID(context.Context, models.User) (models.User, error)
}

type UserService struct {
	repository UserRepository
}

func NewUserService(repository UserRepository) *UserService {
	return &UserService{repository: repository}
}

func (service *UserService) ResolveUser(
	ctx context.Context,
	authlierSubjectID string,
) (models.User, error) {
	authlierSubjectID = strings.TrimSpace(authlierSubjectID)
	if authlierSubjectID == "" {
		return models.User{}, ErrAuthlierSubjectIDRequired
	}

	id, err := ids.New()
	if err != nil {
		return models.User{}, fmt.Errorf("generate user ID: %w", err)
	}

	return service.repository.ResolveUserByAuthlierSubjectID(ctx, models.User{
		ID:                id,
		AuthlierSubjectID: authlierSubjectID,
	})
}
