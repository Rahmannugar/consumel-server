package handlers

import (
	onboardingmodels "github.com/Rahmannugar/consumel-server/internal/onboarding/models"
	"github.com/Rahmannugar/consumel-server/internal/openapi"
)

func setupResponse(setup onboardingmodels.Setup) openapi.OnboardingSetup {
	environments := make([]openapi.ProjectEnvironment, 0, len(setup.Project.Environments))
	for _, environment := range setup.Project.Environments {
		environments = append(environments, openapi.ProjectEnvironment{
			ID: environment.ID.String(), Name: string(environment.Name),
			ActivatedAt: environment.ActivatedAt,
		})
	}
	return openapi.OnboardingSetup{
		Organization: openapi.OnboardingOrganization{
			ID: setup.Organization.ID.String(), Name: setup.Organization.Name,
		},
		Project: openapi.Project{
			ID:               setup.Project.ID.String(),
			OrganizationID:   setup.Organization.ID.String(),
			OrganizationName: setup.Organization.Name,
			Name:             setup.Project.Name,
			Slug:             setup.Project.Slug,
			Environments:     environments,
			CreatedAt:        setup.Project.CreatedAt,
		},
	}
}
