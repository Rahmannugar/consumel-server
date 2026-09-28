package openapi

import customermodels "github.com/Rahmannugar/consumel-server/internal/customers/models"

func customerOperations() []operation {
	customerID := parameter{
		Name: "customerId", Description: "The customer identifier supplied by the integrating application.",
		In: "path", Required: true,
		Schema: map[string]any{"type": "string", "maxLength": customermodels.MaximumCustomerIDLength},
	}
	listParameters := []parameter{
		{Name: "cursor", Description: "The opaque next cursor from the previous page.", In: "query", Schema: map[string]any{"type": "string"}},
		{Name: "limit", Description: "The number of customers to return.", In: "query", Schema: map[string]any{"type": "integer", "minimum": 1, "maximum": 100, "default": 25}},
	}
	dashboardParameters := func(additions ...parameter) []parameter {
		result := append([]parameter{}, projectEnvironmentParameters...)
		return append(result, additions...)
	}

	return []operation{
		{Method: "post", Path: "/v1/customers", Summary: "Create a customer in the API key's project environment.", Request: "CreateCustomerRequest", SuccessCode: "201", Success: "Customer", APIKeyProtected: true, Errors: map[string]string{"400": "CustomerInvalid", "409": "CustomerConflict", "500": "CustomerFailed"}},
		{Method: "get", Path: "/v1/customers", Summary: "List customers in the API key's project environment.", SuccessCode: "200", Success: "Customers", APIKeyProtected: true, Parameters: listParameters, Errors: map[string]string{"400": "CustomerListInvalid", "500": "CustomerFailed"}},
		{Method: "get", Path: "/v1/customers/{customerId}", Summary: "Return one customer from the API key's project environment.", SuccessCode: "200", Success: "Customer", APIKeyProtected: true, Parameters: []parameter{customerID}, Errors: map[string]string{"400": "CustomerInvalid", "404": "CustomerNotFound", "500": "CustomerFailed"}},
		{Method: "put", Path: "/v1/customers/{customerId}", Summary: "Replace a customer's optional contact information and metadata.", Request: "UpdateCustomerRequest", SuccessCode: "200", Success: "Customer", APIKeyProtected: true, Parameters: []parameter{customerID}, Errors: map[string]string{"400": "CustomerInvalid", "404": "CustomerNotFound", "500": "CustomerFailed"}},
		{Method: "post", Path: "/v1/projects/{projectId}/environments/{environment}/customers", Summary: "Create a customer from the signed-in project workspace.", Tag: "Dashboard Customers", Request: "CreateCustomerRequest", SuccessCode: "201", Success: "Customer", Protected: true, Parameters: dashboardParameters(), Errors: map[string]string{"400": "CustomerInvalid", "404": "CustomerEnvironmentNotFound", "409": "CustomerCreateConflict", "500": "CustomerFailed"}},
		{Method: "get", Path: "/v1/projects/{projectId}/environments/{environment}/customers", Summary: "List customers in the signed-in project workspace.", Tag: "Dashboard Customers", SuccessCode: "200", Success: "Customers", Protected: true, Parameters: dashboardParameters(listParameters...), Errors: map[string]string{"400": "CustomerListInvalid", "404": "CustomerEnvironmentNotFound", "409": "CustomerEnvironmentConflict", "500": "CustomerFailed"}},
		{Method: "get", Path: "/v1/projects/{projectId}/environments/{environment}/customers/{customerId}", Summary: "Return one customer in the signed-in project workspace.", Tag: "Dashboard Customers", SuccessCode: "200", Success: "Customer", Protected: true, Parameters: dashboardParameters(customerID), Errors: map[string]string{"400": "CustomerInvalid", "404": "CustomerOrEnvironmentNotFound", "409": "CustomerEnvironmentConflict", "500": "CustomerFailed"}},
		{Method: "put", Path: "/v1/projects/{projectId}/environments/{environment}/customers/{customerId}", Summary: "Replace a customer's optional contact information and supported metadata from the signed-in project workspace.", Tag: "Dashboard Customers", Request: "UpdateCustomerRequest", SuccessCode: "200", Success: "Customer", Protected: true, Parameters: dashboardParameters(customerID), Errors: map[string]string{"400": "CustomerInvalid", "404": "CustomerOrEnvironmentNotFound", "409": "CustomerEnvironmentConflict", "500": "CustomerFailed"}},
	}
}

func customerSchemas() map[string]any {
	optionalMetadata := func() map[string]any {
		return map[string]any{"type": []string{"string", "null"}, "maxLength": customermodels.MaximumMetadataValueLength}
	}
	return map[string]any{
		"CreateCustomerRequest": customermodels.CreateCustomerRequestOpenAPISchema(),
		"UpdateCustomerRequest": customermodels.UpdateCustomerRequestOpenAPISchema(),
		"CustomerMetadata": object([]string{}, map[string]any{
			"plan":     optionalMetadata(),
			"country":  map[string]any{"type": []string{"string", "null"}, "pattern": `^[A-Z]{2}$`, "example": "US"},
			"location": optionalMetadata(),
		}),
		"Customer": object([]string{"customerId", "name", "email", "metadata", "createdAt", "updatedAt"}, map[string]any{
			"customerId": map[string]any{"type": "string", "maxLength": customermodels.MaximumCustomerIDLength},
			"name":       map[string]any{"type": []string{"string", "null"}, "maxLength": customermodels.MaximumCustomerNameLength},
			"email":      map[string]any{"type": []string{"string", "null"}, "format": "email", "maxLength": customermodels.MaximumCustomerEmailLength},
			"metadata":   schemaReference("CustomerMetadata"),
			"createdAt":  map[string]any{"type": "string", "format": "date-time"},
			"updatedAt":  map[string]any{"type": "string", "format": "date-time"},
		}),
		"Customers": object([]string{"customers", "nextCursor"}, map[string]any{
			"customers":  map[string]any{"type": "array", "items": schemaReference("Customer")},
			"nextCursor": map[string]any{"type": []string{"string", "null"}},
		}),
	}
}

func customerErrorResponses() map[string]any {
	return map[string]any{
		"CustomerConflict":              errorResponseWithMessage("The customer ID already exists in the selected environment.", "customer_already_exists", "This customer ID already exists in the selected environment."),
		"CustomerCreateConflict":        errorResponseExamples("The customer cannot be created in the selected environment.", "customer_already_exists", "environment_inactive"),
		"CustomerEnvironmentConflict":   errorResponseWithMessage("The selected project environment is not active.", "environment_inactive", "Activate Live before managing its customers."),
		"CustomerEnvironmentNotFound":   errorResponseWithMessage("The selected project environment is unavailable.", "project_environment_not_found", "This project environment is not available."),
		"CustomerFailed":                errorResponseWithMessage("The customer request could not be completed.", "customer_operation_failed", "Consumel could not complete the customer request. Try again shortly."),
		"CustomerInvalid":               errorResponseExamples("The customer request or identifier is invalid.", "invalid_request", "invalid_customer"),
		"CustomerListInvalid":           errorResponseExamples("The pagination input is invalid.", "invalid_request", "invalid_cursor"),
		"CustomerNotFound":              errorResponseWithMessage("The customer is not present in the authenticated environment.", "customer_not_found", "This customer does not exist in the selected environment."),
		"CustomerOrEnvironmentNotFound": errorResponseExamples("The customer or selected project environment is unavailable.", "customer_not_found", "project_environment_not_found"),
	}
}

func customerExample(name string) (map[string]any, bool) {
	customer := map[string]any{
		"customerId": "user_123", "name": "Jordan Lee", "email": "jordan@example.com",
		"metadata":  map[string]any{"plan": "growth", "country": "US", "location": "New York, NY"},
		"createdAt": "2026-09-28T12:00:00Z", "updatedAt": "2026-09-28T12:00:00Z",
	}
	switch name {
	case "CreateCustomerRequest":
		return map[string]any{
			"customerId": customer["customerId"], "name": customer["name"],
			"email": customer["email"], "metadata": customer["metadata"],
		}, true
	case "UpdateCustomerRequest":
		return map[string]any{
			"name": customer["name"], "email": customer["email"], "metadata": customer["metadata"],
		}, true
	case "Customer":
		return customer, true
	case "Customers":
		return map[string]any{"customers": []any{customer}, "nextCursor": nil}, true
	default:
		return nil, false
	}
}
