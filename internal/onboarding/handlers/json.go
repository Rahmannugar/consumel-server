package handlers

import (
	"time"

	onboardingmodels "github.com/Rahmannugar/consumel-server/internal/onboarding/models"
	projectmodels "github.com/Rahmannugar/consumel-server/internal/projects/models"
)

type responseBody struct {
	Organization organizationResponse `json:"organization"`
	Project      projectResponse      `json:"project"`
}

type organizationResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type projectResponse struct {
	ID               string                `json:"id"`
	OrganizationID   string                `json:"organizationId"`
	OrganizationName string                `json:"organizationName"`
	Name             string                `json:"name"`
	Slug             string                `json:"slug"`
	Environments     []environmentResponse `json:"environments"`
	CreatedAt        time.Time             `json:"createdAt"`
}

type environmentResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	ActivatedAt *time.Time `json:"activatedAt"`
}

func setupResponse(setup onboardingmodels.Setup) responseBody {
	environments := make([]environmentResponse, 0, len(setup.Project.Environments))
	for _, environment := range setup.Project.Environments {
		environments = append(environments, environmentJSON(environment))
	}
	return responseBody{
		Organization: organizationResponse{
			ID: setup.Organization.ID.String(), Name: setup.Organization.Name,
		},
		Project: projectResponse{
			ID:               setup.Project.ID.String(),
			OrganizationID:   setup.Organization.ID.String(),
			OrganizationName: setup.Organization.Name,
			Name:             setup.Project.Name,
			Slug:             setup.Project.Slug,
			Environments:     environments, CreatedAt: setup.Project.CreatedAt,
		},
	}
}

func environmentJSON(environment projectmodels.ProjectEnvironment) environmentResponse {
	return environmentResponse{
		ID: environment.ID.String(), Name: string(environment.Name),
		ActivatedAt: environment.ActivatedAt,
	}
}
