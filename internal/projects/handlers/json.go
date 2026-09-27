package handlers

import (
	"time"

	projectmodels "github.com/Rahmannugar/consumel-server/internal/projects/models"
)

type projectsResponse struct {
	Projects []projectResponse `json:"projects"`
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

func listResponse(projects []projectmodels.Project) projectsResponse {
	result := make([]projectResponse, 0, len(projects))
	for _, project := range projects {
		environments := make([]environmentResponse, 0, len(project.Environments))
		for _, environment := range project.Environments {
			environments = append(environments, environmentResponse{
				ID: environment.ID.String(), Name: string(environment.Name),
				ActivatedAt: environment.ActivatedAt,
			})
		}
		result = append(result, projectResponse{
			ID: project.ID.String(), OrganizationID: project.OrganizationID.String(),
			OrganizationName: project.OrganizationName, Name: project.Name,
			Slug:         project.Slug,
			Environments: environments, CreatedAt: project.CreatedAt,
		})
	}
	return projectsResponse{Projects: result}
}
