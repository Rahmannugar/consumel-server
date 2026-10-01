package models

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

const MaximumProjectNameLength = 120

var (
	ErrProjectNameRequired = errors.New("project name is required")
	ErrProjectNameTooLong  = errors.New("project name is too long")
	ErrProjectNameExists   = errors.New("project name already exists")
)

type CreateProjectRequest struct {
	Name string `json:"name" validate:"required,min=1,max=120" example:"Usage Service"`
}

func (request CreateProjectRequest) Validate() (CreateProjectRequest, error) {
	request.Name = strings.TrimSpace(request.Name)
	switch {
	case request.Name == "":
		return CreateProjectRequest{}, ErrProjectNameRequired
	case len([]rune(request.Name)) > MaximumProjectNameLength:
		return CreateProjectRequest{}, ErrProjectNameTooLong
	default:
		return request, nil
	}
}

type Project struct {
	ID               uuid.UUID
	OrganizationID   uuid.UUID
	OrganizationName string
	Name             string
	Slug             string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Environments     []ProjectEnvironment
}
