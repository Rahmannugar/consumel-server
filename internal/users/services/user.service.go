package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Rahmannugar/consumel-server/internal/common/ids"
	"github.com/Rahmannugar/consumel-server/internal/users/models"
)

var ErrClerkUserIDRequired = errors.New("Clerk user ID is required")

type UserRepository interface {
	CreateUser(context.Context, models.User) (models.User, error)
	UserByClerkID(context.Context, string) (models.User, error)
}

type UserService struct {
	repository UserRepository
}

func NewUserService(repository UserRepository) *UserService {
	return &UserService{repository: repository}
}

func (service *UserService) CreateUser(ctx context.Context, clerkUserID string) (models.User, error) {
	clerkUserID = strings.TrimSpace(clerkUserID)
	if clerkUserID == "" {
		return models.User{}, ErrClerkUserIDRequired
	}

	id, err := ids.New()
	if err != nil {
		return models.User{}, fmt.Errorf("generate user ID: %w", err)
	}

	return service.repository.CreateUser(ctx, models.User{ID: id, ClerkUserID: clerkUserID})
}
