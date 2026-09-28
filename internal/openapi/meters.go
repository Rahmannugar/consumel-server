package openapi

import metermodels "github.com/Rahmannugar/consumel-server/internal/meters/models"

func meterOperations() []operation {
	meterKey := parameter{
		Name: "meterKey", Description: "The stable meter key supplied when the meter was created.",
		In: "path", Required: true,
		Schema: map[string]any{"type": "string", "maxLength": metermodels.MaximumMeterKeyLength, "pattern": `^[a-z][a-z0-9_-]*$`},
	}
	listParameters := []parameter{
		{Name: "cursor", Description: "The opaque next cursor from the previous page.", In: "query", Schema: map[string]any{"type": "string"}},
		{Name: "limit", Description: "The number of meters to return.", In: "query", Schema: map[string]any{"type": "integer", "minimum": 1, "maximum": 100, "default": 25}},
	}
	return []operation{
		{Method: "post", Path: "/v1/meters", Summary: "Create a meter in the API key's project environment.", Request: "CreateMeterRequest", SuccessCode: "201", Success: "Meter", APIKeyProtected: true, Errors: map[string]string{"400": "MeterInvalid", "409": "MeterConflict", "500": "MeterFailed"}},
		{Method: "get", Path: "/v1/meters", Summary: "List meters in the API key's project environment.", SuccessCode: "200", Success: "Meters", APIKeyProtected: true, Parameters: listParameters, Errors: map[string]string{"400": "MeterListInvalid", "500": "MeterFailed"}},
		{Method: "get", Path: "/v1/meters/{meterKey}", Summary: "Return one meter from the API key's project environment.", SuccessCode: "200", Success: "Meter", APIKeyProtected: true, Parameters: []parameter{meterKey}, Errors: map[string]string{"400": "MeterInvalid", "404": "MeterNotFound", "500": "MeterFailed"}},
		{Method: "post", Path: "/v1/projects/{projectId}/environments/{environment}/meters", Summary: "Create a meter from the signed-in project workspace.", Tag: "Dashboard Meters", Request: "CreateMeterRequest", SuccessCode: "201", Success: "Meter", Protected: true, Parameters: dashboardParameters(), Errors: map[string]string{"400": "MeterInvalid", "404": "MeterEnvironmentNotFound", "409": "MeterCreateConflict", "500": "MeterFailed"}},
		{Method: "get", Path: "/v1/projects/{projectId}/environments/{environment}/meters", Summary: "List meters in the signed-in project workspace.", Tag: "Dashboard Meters", SuccessCode: "200", Success: "Meters", Protected: true, Parameters: dashboardParameters(listParameters...), Errors: map[string]string{"400": "MeterListInvalid", "404": "MeterEnvironmentNotFound", "409": "MeterEnvironmentConflict", "500": "MeterFailed"}},
		{Method: "get", Path: "/v1/projects/{projectId}/environments/{environment}/meters/{meterKey}", Summary: "Return one meter from the signed-in project workspace.", Tag: "Dashboard Meters", SuccessCode: "200", Success: "Meter", Protected: true, Parameters: dashboardParameters(meterKey), Errors: map[string]string{"400": "MeterInvalid", "404": "MeterOrEnvironmentNotFound", "409": "MeterEnvironmentConflict", "500": "MeterFailed"}},
	}
}

func meterSchemas() map[string]any {
	return map[string]any{
		"CreateMeterRequest": metermodels.CreateMeterRequestOpenAPISchema(),
		"Meter": object([]string{"id", "meterKey", "name", "description", "type", "createdAt", "updatedAt"}, map[string]any{
			"id":          map[string]any{"type": "string", "format": "uuid"},
			"meterKey":    map[string]any{"type": "string", "maxLength": metermodels.MaximumMeterKeyLength, "pattern": `^[a-z][a-z0-9_-]*$`},
			"name":        map[string]any{"type": "string", "maxLength": metermodels.MaximumMeterNameLength},
			"description": map[string]any{"type": []string{"string", "null"}, "maxLength": metermodels.MaximumMeterDescriptionLength},
			"type":        map[string]any{"type": "string", "enum": []string{"prepaid", "postpaid", "hybrid"}},
			"createdAt":   map[string]any{"type": "string", "format": "date-time"},
			"updatedAt":   map[string]any{"type": "string", "format": "date-time"},
		}),
		"Meters": object([]string{"meters", "nextCursor"}, map[string]any{
			"meters":     map[string]any{"type": "array", "items": schemaReference("Meter")},
			"nextCursor": map[string]any{"type": []string{"string", "null"}},
		}),
	}
}

func meterErrorResponses() map[string]any {
	return map[string]any{
		"MeterConflict":              errorResponseWithMessage("The meter key already exists in the selected environment.", "meter_already_exists", "This meter key already exists in the selected environment."),
		"MeterCreateConflict":        errorResponseExamples("The meter cannot be created in the selected environment.", "meter_already_exists", "environment_inactive"),
		"MeterEnvironmentConflict":   errorResponseWithMessage("The selected project environment is not active.", "environment_inactive", "Activate Live before managing its meters."),
		"MeterEnvironmentNotFound":   errorResponseWithMessage("The selected project environment is unavailable.", "project_environment_not_found", "This project environment is not available."),
		"MeterFailed":                errorResponseWithMessage("The meter request could not be completed.", "meter_operation_failed", "Consumel could not complete the meter request. Try again shortly."),
		"MeterInvalid":               errorResponseExamples("The meter request or key is invalid.", "invalid_request", "invalid_meter"),
		"MeterListInvalid":           errorResponseExamples("The pagination input is invalid.", "invalid_request", "invalid_cursor"),
		"MeterNotFound":              errorResponseWithMessage("The meter is not present in the authenticated environment.", "meter_not_found", "This meter does not exist in the selected environment."),
		"MeterOrEnvironmentNotFound": errorResponseExamples("The meter or selected project environment is unavailable.", "meter_not_found", "project_environment_not_found"),
	}
}

func meterExample(name string) (map[string]any, bool) {
	meter := map[string]any{
		"id": "0199a9f8-f0c4-7f10-90f8-6483353e4624", "meterKey": "api_calls",
		"name": "API calls", "description": "Requests processed by your API.", "type": "postpaid",
		"createdAt": "2026-09-28T12:00:00Z", "updatedAt": "2026-09-28T12:00:00Z",
	}
	switch name {
	case "CreateMeterRequest":
		return map[string]any{"meterKey": meter["meterKey"], "name": meter["name"], "description": meter["description"], "type": meter["type"]}, true
	case "Meter":
		return meter, true
	case "Meters":
		return map[string]any{"meters": []any{meter}, "nextCursor": nil}, true
	default:
		return nil, false
	}
}
