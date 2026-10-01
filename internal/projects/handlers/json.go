package handlers

import (
	"github.com/Rahmannugar/consumel-server/internal/openapi"
	projectmodels "github.com/Rahmannugar/consumel-server/internal/projects/models"
)

func listResponse(projects []projectmodels.Project) openapi.Projects {
	result := make([]openapi.Project, 0, len(projects))
	for _, project := range projects {
		result = append(result, projectJSON(project))
	}
	return openapi.Projects{Projects: result}
}

func projectJSON(project projectmodels.Project) openapi.Project {
	environments := make([]openapi.ProjectEnvironment, 0, len(project.Environments))
	for _, environment := range project.Environments {
		environments = append(environments, openapi.ProjectEnvironment{
			ID: environment.ID.String(), Name: string(environment.Name),
			ActivatedAt: environment.ActivatedAt,
		})
	}
	return openapi.Project{
		ID: project.ID.String(), OrganizationID: project.OrganizationID.String(),
		OrganizationName: project.OrganizationName, Name: project.Name, Slug: project.Slug,
		Environments: environments, CreatedAt: project.CreatedAt,
	}
}

func apiKeyStatus(key *projectmodels.APIKey) openapi.ProjectAPIKeyStatus {
	if key == nil {
		return openapi.ProjectAPIKeyStatus{}
	}
	response := apiKeyJSON(*key)
	return openapi.ProjectAPIKeyStatus{ApiKey: &response}
}

func createdAPIKeyJSON(key projectmodels.CreatedAPIKey) openapi.ProjectAPIKeyCreated {
	return openapi.ProjectAPIKeyCreated{ApiKey: apiKeyJSON(key.APIKey), Secret: key.Secret}
}

func apiKeyJSON(key projectmodels.APIKey) openapi.ProjectAPIKey {
	return openapi.ProjectAPIKey{
		ID: key.ID, Environment: string(key.Environment),
		Prefix: key.Prefix, LastFour: key.LastFour,
		CreatedAt: key.CreatedAt, LastUsedAt: key.LastUsedAt,
	}
}
