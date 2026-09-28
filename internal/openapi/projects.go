package openapi

import projectmodels "github.com/Rahmannugar/consumel-server/internal/projects/models"

var projectEnvironmentParameters = []parameter{
	{Name: "projectId", Description: "The immutable ID of the project selected in the dashboard.", In: "path", Required: true, Schema: map[string]any{"type": "string", "format": "uuid"}},
	{Name: "environment", Description: "The selected isolated project environment.", In: "path", Required: true, Schema: map[string]any{"type": "string", "enum": []string{"sandbox", "live"}}},
}

func dashboardParameters(additions ...parameter) []parameter {
	result := append([]parameter{}, projectEnvironmentParameters...)
	return append(result, additions...)
}

func projectOperations() []operation {
	return []operation{
		{Method: "get", Path: "/v1/projects", Summary: "Return the signed-in user's active projects and environments.", SuccessCode: "200", Success: "Projects", Protected: true, Errors: map[string]string{"500": "ProjectsLoadFailed"}},
		{Method: "post", Path: "/v1/projects", Summary: "Create a project with isolated Sandbox and Live environments.", Request: "CreateProjectRequest", SuccessCode: "201", Success: "Project", Protected: true, Errors: map[string]string{"400": "ProjectCreateInvalid", "409": "ProjectCreateConflict", "500": "ProjectCreateFailed"}},
		{Method: "get", Path: "/v1/projects/{projectId}/environments/{environment}/api-key", Summary: "Return safe metadata for the environment's active API key.", SuccessCode: "200", Success: "ProjectAPIKeyStatus", Protected: true, Parameters: projectEnvironmentParameters, Errors: map[string]string{"400": "ProjectAPIKeyInvalid", "404": "ProjectAPIKeyNotFound", "500": "ProjectAPIKeyFailed"}},
		{Method: "post", Path: "/v1/projects/{projectId}/environments/{environment}/api-key", Summary: "Create the environment's first active API key and reveal its plaintext once.", SuccessCode: "201", Success: "ProjectAPIKeyCreated", Protected: true, Parameters: projectEnvironmentParameters, Errors: map[string]string{"400": "ProjectAPIKeyInvalid", "404": "ProjectAPIKeyNotFound", "409": "ProjectAPIKeyConflict", "500": "ProjectAPIKeyFailed"}},
		{Method: "delete", Path: "/v1/projects/{projectId}/environments/{environment}/api-key", Summary: "Revoke the environment's active API key.", SuccessCode: "204", Protected: true, Parameters: projectEnvironmentParameters, Errors: map[string]string{"400": "ProjectAPIKeyInvalid", "404": "ProjectAPIKeyNotFound", "500": "ProjectAPIKeyFailed"}},
		{Method: "post", Path: "/v1/projects/{projectId}/environments/{environment}/api-key/replace", Summary: "Revoke the active API key and reveal its replacement once.", SuccessCode: "201", Success: "ProjectAPIKeyCreated", Protected: true, Parameters: projectEnvironmentParameters, Errors: map[string]string{"400": "ProjectAPIKeyInvalid", "404": "ProjectAPIKeyNotFound", "409": "ProjectAPIKeyConflict", "500": "ProjectAPIKeyFailed"}},
		{Method: "post", Path: "/v1/projects/{projectId}/environments/{environment}/activate", Summary: "Activate the selected project environment explicitly.", SuccessCode: "200", Success: "ProjectEnvironment", Protected: true, Parameters: projectEnvironmentParameters, Errors: map[string]string{"400": "ProjectAPIKeyInvalid", "404": "ProjectAPIKeyNotFound", "500": "ProjectAPIKeyFailed"}},
	}
}

func projectSchemas() map[string]any {
	stringProperty := func() map[string]any { return map[string]any{"type": "string"} }
	return map[string]any{
		"CreateProjectRequest": projectmodels.CreateProjectRequestOpenAPISchema(),
		"Project": object([]string{"id", "organizationId", "organizationName", "name", "slug", "environments", "createdAt"}, map[string]any{
			"id": stringProperty(), "organizationId": stringProperty(), "organizationName": stringProperty(), "name": stringProperty(),
			"slug":         map[string]any{"type": "string", "minLength": 1, "maxLength": 120, "example": "acme-api"},
			"environments": map[string]any{"type": "array", "minItems": 2, "maxItems": 2, "items": schemaReference("ProjectEnvironment")},
			"createdAt":    map[string]any{"type": "string", "format": "date-time"},
		}),
		"ProjectEnvironment": object([]string{"id", "name", "activatedAt"}, map[string]any{
			"id": stringProperty(), "name": map[string]any{"type": "string", "enum": []string{"sandbox", "live"}},
			"activatedAt": map[string]any{"type": []string{"string", "null"}, "format": "date-time"},
		}),
		"Projects": object([]string{"projects"}, map[string]any{"projects": map[string]any{"type": "array", "items": schemaReference("Project")}}),
		"ProjectAPIKey": object([]string{"id", "environment", "prefix", "lastFour", "createdAt", "lastUsedAt"}, map[string]any{
			"id": map[string]any{"type": "string", "format": "uuid"}, "environment": map[string]any{"type": "string", "enum": []string{"sandbox", "live"}},
			"prefix":    map[string]any{"type": "string", "enum": []string{"cm_test_", "cm_live_"}},
			"lastFour":  map[string]any{"type": "string", "minLength": 4, "maxLength": 4, "example": "QBY0"},
			"createdAt": map[string]any{"type": "string", "format": "date-time"}, "lastUsedAt": map[string]any{"type": []string{"string", "null"}, "format": "date-time"},
		}),
		"ProjectAPIKeyStatus": object([]string{"apiKey"}, map[string]any{"apiKey": map[string]any{"oneOf": []map[string]any{schemaReference("ProjectAPIKey"), {"type": "null"}}}}),
		"ProjectAPIKeyCreated": object([]string{"apiKey", "secret"}, map[string]any{
			"apiKey": schemaReference("ProjectAPIKey"),
			"secret": map[string]any{"type": "string", "writeOnly": true, "pattern": `^cm_(test|live)_[A-Za-z0-9_-]{43}$`, "example": "cm_test_3xKq7VfJm2zY8wN4aBcD6eFgH9iLpQrStUvWx0Z1A2B"},
		}),
	}
}

func projectErrorResponses() map[string]any {
	return map[string]any{
		"ProjectsLoadFailed":    errorResponseWithMessage("The signed-in user's projects could not be loaded.", "projects_load_failed", "Consumel could not load your projects. Try again shortly."),
		"ProjectCreateInvalid":  errorResponseWithMessage("The project name is invalid.", "invalid_request", "Enter a project name between 1 and 120 characters."),
		"ProjectCreateConflict": errorResponseExamples("The project cannot be created in the current organization state.", "project_name_exists", "organization_required"),
		"ProjectCreateFailed":   errorResponseWithMessage("The project could not be created.", "project_create_failed", "Consumel could not create the project. Try again shortly."),
		"ProjectAPIKeyInvalid":  errorResponseWithMessage("The project ID or environment is invalid.", "invalid_request", "Choose a valid project and environment."),
		"ProjectAPIKeyNotFound": errorResponseExamples("The project environment or active API key is unavailable.", "project_environment_not_found", "api_key_not_found"),
		"ProjectAPIKeyConflict": errorResponseExamples("The requested API key action conflicts with the environment state.", "environment_inactive", "api_key_already_exists"),
		"ProjectAPIKeyFailed":   errorResponseWithMessage("The API key action could not be completed.", "api_key_operation_failed", "Consumel could not complete the API key action. Try again shortly."),
	}
}

func projectExampleFor(name string) (map[string]any, bool) {
	switch name {
	case "CreateProjectRequest":
		return map[string]any{"name": "Usage Service"}, true
	case "Projects":
		return map[string]any{"projects": []any{projectExample()}}, true
	case "Project":
		return projectExample(), true
	case "ProjectEnvironment":
		return projectExample()["environments"].([]any)[0].(map[string]any), true
	case "ProjectAPIKeyStatus":
		return map[string]any{"apiKey": apiKeyExample()}, true
	case "ProjectAPIKeyCreated":
		return map[string]any{"apiKey": apiKeyExample(), "secret": "cm_test_3xKq7VfJm2zY8wN4aBcD6eFgH9iLpQrStUvWx0Z1A2B"}, true
	default:
		return nil, false
	}
}

func apiKeyExample() map[string]any {
	return map[string]any{"id": "0199a7e1-8f18-7b6e-90c9-dc7b4ace22d1", "environment": "sandbox", "prefix": "cm_test_", "lastFour": "Z1A2", "createdAt": "2026-09-27T12:00:00Z", "lastUsedAt": nil}
}

func projectExample() map[string]any {
	return map[string]any{
		"id": "0199a417-05da-7aa2-b024-2011f24972da", "organizationId": "0199a416-d2c8-75ea-bdb4-1d13c627169b",
		"organizationName": "Acme", "name": "Acme API", "slug": "acme-api", "createdAt": "2026-09-25T12:08:00Z",
		"environments": []any{
			map[string]any{"id": "0199a417-1ae1-7b67-ad5b-809be2f9ca0a", "name": "sandbox", "activatedAt": "2026-09-25T12:08:00Z"},
			map[string]any{"id": "0199a417-30d7-7ccc-978b-ec14f67a4d14", "name": "live", "activatedAt": nil},
		},
	}
}
