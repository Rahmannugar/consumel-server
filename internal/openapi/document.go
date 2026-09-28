package openapi

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type operation struct {
	Method, Path, Summary, Tag, Request, SuccessCode, Success, SuccessDescription string
	AlternateSuccess, Errors                                                      map[string]string
	Protected, APIKeyProtected                                                    bool
	Parameters                                                                    []parameter
}

type parameter struct {
	Name, Description, In string
	Required              bool
	Schema                map[string]any
}

// Document builds the deterministic OpenAPI contract served by the API and
// written to openapi.json for review and stale-output checks.
func Document() ([]byte, error) {
	operations := allOperations()
	sort.Slice(operations, func(i, j int) bool {
		if operations[i].Path == operations[j].Path {
			return operations[i].Method < operations[j].Method
		}
		return operations[i].Path < operations[j].Path
	})
	paths := map[string]any{}
	for _, endpoint := range operations {
		path, _ := paths[endpoint.Path].(map[string]any)
		if path == nil {
			path = map[string]any{}
			paths[endpoint.Path] = path
		}
		description := endpoint.SuccessDescription
		if description == "" {
			description = successDescription(endpoint.SuccessCode)
		}
		success := map[string]any{"description": description}
		if endpoint.Success != "" {
			success["content"] = jsonContent(schemaReference(endpoint.Success), exampleFor(endpoint.Success))
		}
		responses := operationResponses(endpoint, success)
		for code, alternateDescription := range endpoint.AlternateSuccess {
			alternate := map[string]any{"description": alternateDescription}
			if endpoint.Success != "" {
				alternate["content"] = jsonContent(schemaReference(endpoint.Success), exampleFor(endpoint.Success))
			}
			responses[code] = alternate
		}
		tag := endpoint.Tag
		if tag == "" {
			tag = tagFor(endpoint.Path)
		}
		operationDocument := map[string]any{"summary": endpoint.Summary, "tags": []string{tag}, "responses": responses}
		if endpoint.Request != "" {
			operationDocument["requestBody"] = map[string]any{"required": true, "content": jsonContent(schemaReference(endpoint.Request), exampleFor(endpoint.Request))}
		}
		if len(endpoint.Parameters) > 0 {
			parameters := make([]map[string]any, 0, len(endpoint.Parameters))
			for _, value := range endpoint.Parameters {
				parameters = append(parameters, map[string]any{"name": value.Name, "in": value.In, "required": value.Required, "description": value.Description, "schema": value.Schema})
			}
			operationDocument["parameters"] = parameters
		}
		if endpoint.Protected {
			operationDocument["security"] = []map[string][]string{{"productionCookieSession": {}}, {"localCookieSession": {}}}
		}
		if endpoint.APIKeyProtected {
			operationDocument["security"] = []map[string][]string{{"projectAPIKey": {}}}
		}
		path[endpoint.Method] = operationDocument
	}
	document := map[string]any{
		"openapi": "3.1.0",
		"info":    map[string]any{"title": "Consumel API", "version": "0.1.0", "description": "Consumel usage-based billing infrastructure API."},
		"servers": []map[string]string{{"url": "https://api.consumel.com", "description": "Production"}, {"url": "http://localhost:8080", "description": "Local development"}},
		"paths":   paths,
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"productionCookieSession": map[string]any{"type": "apiKey", "in": "cookie", "name": "__Host-consumel_session"},
				"localCookieSession":      map[string]any{"type": "apiKey", "in": "cookie", "name": "consumel_session"},
				"projectAPIKey":           map[string]any{"type": "http", "scheme": "bearer", "bearerFormat": "cm_test_… or cm_live_…"},
			},
			"schemas": schemas(), "responses": errorResponses(),
		},
	}
	encoded, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode OpenAPI document: %w", err)
	}
	return append(encoded, '\n'), nil
}

func allOperations() []operation {
	groups := [][]operation{authenticationOperations(), healthOperations(), onboardingOperations(), projectOperations(), customerOperations(), meterOperations()}
	var result []operation
	for _, group := range groups {
		result = append(result, group...)
	}
	return result
}

func schemas() map[string]any {
	result := map[string]any{"Error": object([]string{"error"}, map[string]any{"error": object([]string{"code"}, map[string]any{"code": map[string]any{"type": "string"}, "message": map[string]any{"type": "string"}})})}
	for _, additions := range []map[string]any{authenticationSchemas(), healthSchemas(), onboardingSchemas(), projectSchemas(), customerSchemas(), meterSchemas()} {
		mergeComponents(result, additions)
	}
	return result
}

func errorResponses() map[string]any {
	result := map[string]any{"ServerError": errorResponse("The request could not be completed.", "authentication_failed")}
	for _, additions := range []map[string]any{authenticationErrorResponses(), onboardingErrorResponses(), projectErrorResponses(), customerErrorResponses(), meterErrorResponses()} {
		mergeComponents(result, additions)
	}
	return result
}

func operationResponses(endpoint operation, success map[string]any) map[string]any {
	responses := map[string]any{endpoint.SuccessCode: success, "500": responseReference("ServerError")}
	if endpoint.Request != "" || endpoint.Path == "/auth/google/callback" {
		responses["400"] = responseReference("BadRequest")
	}
	if endpoint.Protected {
		responses["401"] = responseReference("NotAuthenticated")
	}
	if endpoint.APIKeyProtected {
		responses["401"] = responseReference("InvalidAPIKey")
	}
	if tagFor(endpoint.Path) == "Authentication" {
		responses["429"] = responseReference("RateLimited")
	}
	if endpoint.Path == "/health/ready" {
		responses["503"] = map[string]any{"description": "PostgreSQL is unavailable.", "content": jsonContent(schemaReference("Health"), map[string]any{"status": "unavailable"})}
	}
	for status, response := range endpoint.Errors {
		responses[status] = responseReference(response)
	}
	return responses
}

func exampleFor(name string) map[string]any {
	providers := []func(string) (map[string]any, bool){authenticationExample, healthExample, onboardingExample, projectExampleFor, customerExample, meterExample}
	for _, provider := range providers {
		if example, ok := provider(name); ok {
			return example
		}
	}
	return map[string]any{}
}

func tagFor(path string) string {
	switch {
	case strings.HasPrefix(path, "/health"):
		return "Health"
	case path == "/onboarding":
		return "Onboarding"
	case strings.Contains(path, "/customers"):
		return "Customers"
	case strings.Contains(path, "/meters"):
		return "Meters"
	case strings.HasPrefix(path, "/v1/projects"):
		return "Projects"
	default:
		return "Authentication"
	}
}

func errorResponse(description, code string) map[string]any {
	return map[string]any{"description": description, "content": jsonContent(schemaReference("Error"), map[string]any{"error": map[string]any{"code": code}})}
}
func errorResponseWithMessage(description, code, message string) map[string]any {
	return map[string]any{"description": description, "content": jsonContent(schemaReference("Error"), map[string]any{"error": map[string]any{"code": code, "message": message}})}
}
func errorResponseExamples(description string, codes ...string) map[string]any {
	examples := make(map[string]any, len(codes))
	for _, code := range codes {
		examples[code] = map[string]any{"value": map[string]any{"error": map[string]any{"code": code}}}
	}
	return map[string]any{"description": description, "content": map[string]any{"application/json": map[string]any{"schema": schemaReference("Error"), "examples": examples}}}
}
func responseReference(name string) map[string]any {
	return map[string]any{"$ref": "#/components/responses/" + name}
}
func schemaReference(name string) map[string]any {
	return map[string]any{"$ref": "#/components/schemas/" + name}
}
func jsonContent(schema, example map[string]any) map[string]any {
	return map[string]any{"application/json": map[string]any{"schema": schema, "example": example}}
}
func object(required []string, properties map[string]any) map[string]any {
	return map[string]any{"type": "object", "additionalProperties": false, "required": required, "properties": properties}
}
func mergeComponents(target, additions map[string]any) {
	for name, component := range additions {
		target[name] = component
	}
}
func successDescription(status string) string {
	switch status {
	case "201":
		return "Created."
	case "202":
		return "Accepted without revealing whether the account exists."
	case "204":
		return "Completed without a response body."
	case "303":
		return "Redirects to the client after authentication."
	default:
		return "Completed successfully."
	}
}
