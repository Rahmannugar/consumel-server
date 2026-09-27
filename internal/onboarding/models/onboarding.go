package models

import (
	"errors"
	"strings"

	organizationmodels "github.com/Rahmannugar/consumel-server/internal/organizations/models"
	projectmodels "github.com/Rahmannugar/consumel-server/internal/projects/models"
)

const MaximumNameLength = 120

var (
	ErrOrganizationNameRequired           = errors.New("organization name is required")
	ErrOrganizationNameTooLong            = errors.New("organization name is too long")
	ErrProjectNameRequired                = errors.New("project name is required")
	ErrProjectNameTooLong                 = errors.New("project name is too long")
	ErrExistingOrganizationWithoutProject = errors.New(
		"account already belongs to an organization without a project",
	)
)

type SetupRequest struct {
	OrganizationName string `json:"organizationName"`
	ProjectName      string `json:"projectName"`
}

func (request SetupRequest) Validate() (SetupRequest, error) {
	request.OrganizationName = strings.TrimSpace(request.OrganizationName)
	request.ProjectName = strings.TrimSpace(request.ProjectName)
	switch {
	case request.OrganizationName == "":
		return SetupRequest{}, ErrOrganizationNameRequired
	case len([]rune(request.OrganizationName)) > MaximumNameLength:
		return SetupRequest{}, ErrOrganizationNameTooLong
	case request.ProjectName == "":
		return SetupRequest{}, ErrProjectNameRequired
	case len([]rune(request.ProjectName)) > MaximumNameLength:
		return SetupRequest{}, ErrProjectNameTooLong
	default:
		return request, nil
	}
}

func SetupRequestOpenAPISchema() map[string]any {
	name := func() map[string]any {
		return map[string]any{"type": "string", "minLength": 1, "maxLength": MaximumNameLength}
	}
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"organizationName", "projectName"},
		"properties": map[string]any{
			"organizationName": name(),
			"projectName":      name(),
		},
	}
}

type Setup struct {
	Organization organizationmodels.Organization
	Project      projectmodels.Project
	Created      bool
}
