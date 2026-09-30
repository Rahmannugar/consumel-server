package openapi

import (
	consumptionmodels "github.com/Rahmannugar/consumel-server/internal/core/consumption/models"
	customermodels "github.com/Rahmannugar/consumel-server/internal/customers/models"
	metermodels "github.com/Rahmannugar/consumel-server/internal/meters/models"
)

func consumptionOperations() []operation {
	customerID := parameter{
		Name: "customerId", Description: "The customer identifier supplied by the integrating application.",
		In: "path", Required: true, Schema: map[string]any{"type": "string", "maxLength": customermodels.MaximumCustomerIDLength},
	}
	meterKey := parameter{
		Name: "meterKey", Description: "The stable meter key supplied when the meter was created.",
		In: "path", Required: true,
		Schema: map[string]any{"type": "string", "maxLength": metermodels.MaximumMeterKeyLength, "pattern": `^[a-z][a-z0-9_-]*$`},
	}
	idempotencyKey := parameter{
		Name: "Idempotency-Key", Description: "A UUID v7 that identifies this logical balance addition. Reuse it only when retrying the same request.",
		In: "header", Required: true, Schema: map[string]any{"type": "string", "format": "uuid"},
	}
	consumeIdempotencyKey := parameter{
		Name: "Idempotency-Key", Description: "A UUID v7 that identifies this logical consume operation. Reuse it only when retrying the same request.",
		In: "header", Required: true, Schema: map[string]any{"type": "string", "format": "uuid"},
	}
	replayHeader := map[string]any{
		"Idempotency-Replayed": map[string]any{
			"description": "Present with the value true when Consumel returns the original result of a committed retry.",
			"schema":      map[string]any{"type": "string", "enum": []string{"true"}},
		},
	}
	publicReadErrors := map[string]string{"400": "BalanceInvalid", "404": "BalanceNotFound", "500": "BalanceFailed"}
	dashboardReadErrors := map[string]string{"400": "BalanceInvalid", "404": "BalanceOrEnvironmentNotFound", "409": "BalanceEnvironmentConflict", "500": "BalanceFailed"}
	publicAddErrors := map[string]string{"400": "BalanceInvalid", "404": "BalanceSubjectNotFound", "409": "BalanceConflict", "500": "BalanceFailed"}
	dashboardAddErrors := map[string]string{"400": "BalanceInvalid", "404": "BalanceSubjectOrEnvironmentNotFound", "409": "BalanceOrEnvironmentConflict", "500": "BalanceFailed"}
	publicSetErrors := map[string]string{"400": "BalanceInvalid", "404": "BalanceSubjectNotFound", "500": "BalanceFailed"}
	dashboardSetErrors := map[string]string{"400": "BalanceInvalid", "404": "BalanceSubjectOrEnvironmentNotFound", "409": "BalanceEnvironmentConflict", "500": "BalanceFailed"}
	return []operation{
		{Method: "post", Path: "/v1/consume", Summary: "Atomically record usage and apply the active meter's balance behavior.", Tag: "Consumption", Request: "ConsumeRequest", SuccessCode: "200", Success: "UsageEvent", SuccessHeaders: replayHeader, APIKeyProtected: true, Parameters: []parameter{consumeIdempotencyKey}, Errors: map[string]string{"400": "ConsumeInvalid", "404": "ConsumeMeterNotFound", "409": "ConsumeConflict", "500": "ConsumeFailed"}},
		{Method: "get", Path: "/v1/projects/{projectId}/environments/{environment}/events", Summary: "List accepted and denied usage operations in the signed-in project workspace.", Tag: "Dashboard Events", SuccessCode: "200", Success: "UsageOperations", Protected: true, Parameters: dashboardParameters(
			parameter{Name: "status", Description: "Filter by the persisted operation outcome.", In: "query", Schema: map[string]any{"type": "string", "enum": []string{"accepted", "denied"}}},
			parameter{Name: "from", Description: "Inclusive RFC 3339 start time. Supply from and to together; the range may span at most one year.", In: "query", Schema: map[string]any{"type": "string", "format": "date-time"}},
			parameter{Name: "to", Description: "Exclusive RFC 3339 end time. Supply from and to together.", In: "query", Schema: map[string]any{"type": "string", "format": "date-time"}},
			parameter{Name: "cursor", Description: "The opaque next cursor from the previous page.", In: "query", Schema: map[string]any{"type": "string"}},
			parameter{Name: "limit", Description: "The number of events to return.", In: "query", Schema: map[string]any{"type": "integer", "minimum": 1, "maximum": 100, "default": 50}},
		), Errors: map[string]string{"400": "OperationListInvalid", "404": "OperationEnvironmentNotFound", "409": "OperationEnvironmentConflict", "500": "OperationListFailed"}},
		{Method: "get", Path: "/v1/projects/{projectId}/environments/{environment}/events/stream", Summary: "Stream newly persisted usage operations to the signed-in project workspace.", Tag: "Dashboard Events", SuccessCode: "200", SuccessDescription: "A server-sent event stream. Each usage.operation event contains a UsageOperation JSON payload and a Redis stream cursor used for reconnection.", SuccessMediaType: "text/event-stream", Protected: true, Parameters: dashboardParameters(
			parameter{Name: "cursor", Description: "The last Redis stream cursor received by the client. Browsers normally reconnect through Last-Event-ID automatically.", In: "query", Schema: map[string]any{"type": "string", "pattern": `^[0-9]+-[0-9]+$`}},
		), Errors: map[string]string{"400": "OperationListInvalid", "404": "OperationEnvironmentNotFound", "409": "OperationEnvironmentConflict", "500": "OperationListFailed"}},
		{Method: "get", Path: "/v1/customers/{customerId}/balances", Summary: "List a customer's active meter balances in the API key's project environment.", Tag: "Balances", SuccessCode: "200", Success: "Balances", APIKeyProtected: true, Parameters: []parameter{customerID}, Errors: publicReadErrors},
		{Method: "get", Path: "/v1/customers/{customerId}/balances/{meterKey}", Summary: "Return one customer and meter balance from the API key's project environment.", Tag: "Balances", SuccessCode: "200", Success: "Balance", APIKeyProtected: true, Parameters: []parameter{customerID, meterKey}, Errors: publicReadErrors},
		{Method: "post", Path: "/v1/balances", Summary: "Add an optionally expiring, idempotent quantity to a customer and meter balance.", Tag: "Balances", Request: "AddBalanceRequest", SuccessCode: "200", Success: "Balance", SuccessHeaders: replayHeader, APIKeyProtected: true, Parameters: []parameter{idempotencyKey}, Errors: publicAddErrors},
		{Method: "put", Path: "/v1/balances/{customerId}/{meterKey}", Summary: "Set a customer and meter balance to an exact quantity.", Tag: "Balances", Request: "SetBalanceRequest", SuccessCode: "200", Success: "Balance", APIKeyProtected: true, Parameters: []parameter{customerID, meterKey}, Errors: publicSetErrors},
		{Method: "get", Path: "/v1/projects/{projectId}/environments/{environment}/customers/{customerId}/balances", Summary: "List a customer's active meter balances from the signed-in project workspace.", Tag: "Dashboard Balances", SuccessCode: "200", Success: "Balances", Protected: true, Parameters: dashboardParameters(customerID), Errors: dashboardReadErrors},
		{Method: "get", Path: "/v1/projects/{projectId}/environments/{environment}/customers/{customerId}/balances/{meterKey}", Summary: "Return one customer and meter balance from the signed-in project workspace.", Tag: "Dashboard Balances", SuccessCode: "200", Success: "Balance", Protected: true, Parameters: dashboardParameters(customerID, meterKey), Errors: dashboardReadErrors},
		{Method: "post", Path: "/v1/projects/{projectId}/environments/{environment}/balances", Summary: "Add an optionally expiring, idempotent quantity to a customer and meter balance from the signed-in project workspace.", Tag: "Dashboard Balances", Request: "AddBalanceRequest", SuccessCode: "200", Success: "Balance", SuccessHeaders: replayHeader, Protected: true, Parameters: dashboardParameters(idempotencyKey), Errors: dashboardAddErrors},
		{Method: "put", Path: "/v1/projects/{projectId}/environments/{environment}/balances/{customerId}/{meterKey}", Summary: "Set a customer and meter balance to an exact quantity from the signed-in project workspace.", Tag: "Dashboard Balances", Request: "SetBalanceRequest", SuccessCode: "200", Success: "Balance", Protected: true, Parameters: dashboardParameters(customerID, meterKey), Errors: dashboardSetErrors},
	}
}

func consumptionSchemas() map[string]any {
	return map[string]any{
		"ConsumeRequest":    consumptionmodels.ConsumeRequestOpenAPISchema(),
		"AddBalanceRequest": consumptionmodels.AddBalanceRequestOpenAPISchema(),
		"SetBalanceRequest": consumptionmodels.SetBalanceRequestOpenAPISchema(),
		"Balance": object([]string{"id", "customerId", "meterKey", "quantity", "nextExpiresAt", "createdAt", "updatedAt"}, map[string]any{
			"id":            map[string]any{"type": "string", "format": "uuid"},
			"customerId":    map[string]any{"type": "string", "maxLength": customermodels.MaximumCustomerIDLength},
			"meterKey":      map[string]any{"type": "string", "maxLength": metermodels.MaximumMeterKeyLength, "pattern": `^[a-z][a-z0-9_-]*$`},
			"quantity":      map[string]any{"type": "integer", "format": "int64", "minimum": 0},
			"nextExpiresAt": map[string]any{"type": []string{"string", "null"}, "format": "date-time"},
			"createdAt":     map[string]any{"type": "string", "format": "date-time"},
			"updatedAt":     map[string]any{"type": "string", "format": "date-time"},
		}),
		"Balances": object([]string{"balances"}, map[string]any{
			"balances": map[string]any{"type": "array", "items": schemaReference("Balance")},
		}),
		"UsageEvent": object([]string{"id", "customerId", "meterKey", "quantity", "meterType", "balanceDebited", "remainingBalance", "billable", "createdAt"}, map[string]any{
			"id":               map[string]any{"type": "string", "format": "uuid"},
			"customerId":       map[string]any{"type": "string", "maxLength": customermodels.MaximumCustomerIDLength},
			"meterKey":         map[string]any{"type": "string", "maxLength": metermodels.MaximumMeterKeyLength, "pattern": `^[a-z][a-z0-9_-]*$`},
			"quantity":         map[string]any{"type": "integer", "format": "int64", "minimum": 1},
			"meterType":        map[string]any{"type": "string", "enum": []string{"prepaid", "postpaid", "hybrid"}},
			"balanceDebited":   map[string]any{"type": "integer", "format": "int64", "minimum": 0},
			"remainingBalance": map[string]any{"type": []string{"integer", "null"}, "format": "int64", "minimum": 0},
			"billable":         map[string]any{"type": "boolean"},
			"createdAt":        map[string]any{"type": "string", "format": "date-time"},
		}),
		"UsageOperation": object([]string{"id", "customerId", "meterKey", "quantity", "meterType", "status", "denialReason", "balanceDebited", "remainingBalance", "billable", "replayCount", "lastReplayedAt", "createdAt"}, map[string]any{
			"id": map[string]any{"type": "string", "format": "uuid"}, "customerId": map[string]any{"type": "string"},
			"meterKey": map[string]any{"type": "string"}, "quantity": map[string]any{"type": "integer", "format": "int64", "minimum": 1},
			"meterType":    map[string]any{"type": "string", "enum": []string{"prepaid", "postpaid", "hybrid"}},
			"status":       map[string]any{"type": "string", "enum": []string{"accepted", "denied"}},
			"denialReason": map[string]any{"type": []string{"string", "null"}}, "balanceDebited": map[string]any{"type": "integer", "format": "int64", "minimum": 0},
			"remainingBalance": map[string]any{"type": []string{"integer", "null"}, "format": "int64", "minimum": 0},
			"billable":         map[string]any{"type": "boolean"}, "replayCount": map[string]any{"type": "integer", "format": "int64", "minimum": 0},
			"lastReplayedAt": map[string]any{"type": []string{"string", "null"}, "format": "date-time"}, "createdAt": map[string]any{"type": "string", "format": "date-time"},
		}),
		"UsageOperations": object([]string{"operations", "nextCursor"}, map[string]any{
			"operations": map[string]any{"type": "array", "items": schemaReference("UsageOperation")},
			"nextCursor": map[string]any{"type": []string{"string", "null"}},
		}),
	}
}

func consumptionErrorResponses() map[string]any {
	return map[string]any{
		"ConsumeConflict":                     errorResponseExamples("The consume operation conflicts with the current balance or prior idempotent request.", "insufficient_balance", "idempotency_key_conflict"),
		"ConsumeFailed":                       errorResponseWithMessage("The consume operation could not be completed.", "consume_failed", "Consumel could not process this usage. Try again shortly."),
		"ConsumeInvalid":                      errorResponseWithMessage("The consume request or idempotency key is invalid.", "invalid_consume", "Check the customer ID, meter key, quantity, and idempotency key."),
		"ConsumeMeterNotFound":                errorResponseWithMessage("The active meter is unavailable.", "meter_not_found", "The active meter does not exist in this environment."),
		"OperationEnvironmentConflict":        errorResponseWithMessage("The selected project environment is not active.", "environment_inactive", "Activate Live before viewing its events."),
		"OperationEnvironmentNotFound":        errorResponseWithMessage("The selected project environment is unavailable.", "project_environment_not_found", "This project environment is not available."),
		"OperationListFailed":                 errorResponseWithMessage("The event list could not be loaded.", "operation_list_failed", "Consumel could not load events. Try again shortly."),
		"OperationListInvalid":                errorResponseExamples("The event list filter or pagination input is invalid.", "invalid_request", "invalid_cursor", "invalid_filter"),
		"BalanceConflict":                     errorResponseExamples("The balance addition conflicts with prior state.", "idempotency_key_conflict", "balance_limit_exceeded"),
		"BalanceEnvironmentConflict":          errorResponseWithMessage("The selected project environment is not active.", "environment_inactive", "Activate Live before managing its balances."),
		"BalanceFailed":                       errorResponseWithMessage("The balance request could not be completed.", "balance_operation_failed", "Consumel could not complete the balance request. Try again shortly."),
		"BalanceInvalid":                      errorResponseExamples("The balance request, resource key, or idempotency key is invalid.", "invalid_request", "invalid_balance"),
		"BalanceNotFound":                     errorResponseExamples("The customer, active meter, or balance is unavailable.", "balance_subject_not_found", "balance_not_found"),
		"BalanceOrEnvironmentConflict":        errorResponseExamples("The balance addition conflicts with prior state or the selected environment is inactive.", "idempotency_key_conflict", "balance_limit_exceeded", "environment_inactive"),
		"BalanceOrEnvironmentNotFound":        errorResponseExamples("The customer, active meter, balance, or selected environment is unavailable.", "balance_subject_not_found", "balance_not_found", "project_environment_not_found"),
		"BalanceSubjectNotFound":              errorResponseWithMessage("The customer or active meter is unavailable.", "balance_subject_not_found", "The customer or active meter does not exist in this environment."),
		"BalanceSubjectOrEnvironmentNotFound": errorResponseExamples("The customer, active meter, or selected project environment is unavailable.", "balance_subject_not_found", "project_environment_not_found"),
	}
}

func consumptionExample(name string) (map[string]any, bool) {
	balance := map[string]any{
		"id": "0199aa81-ce8c-73bf-a880-8e84654b9a6c", "customerId": "user_123", "meterKey": "api_calls",
		"quantity": 10000, "nextExpiresAt": "2026-10-31T00:00:00Z", "createdAt": "2026-09-28T12:00:00Z", "updatedAt": "2026-09-28T12:00:00Z",
	}
	switch name {
	case "ConsumeRequest":
		return map[string]any{"customerId": "customer_123", "meterKey": "api_calls", "quantity": 500}, true
	case "AddBalanceRequest":
		return map[string]any{"customerId": "user_123", "meterKey": "api_calls", "quantity": 10000, "expiresAt": "2026-10-31T00:00:00Z"}, true
	case "SetBalanceRequest":
		return map[string]any{"quantity": 10000}, true
	case "Balance":
		return balance, true
	case "Balances":
		return map[string]any{"balances": []any{balance}}, true
	case "UsageEvent":
		return map[string]any{
			"id": "0199aad1-f00d-7ae0-935f-cde0bd9f3ba5", "customerId": "customer_123",
			"meterKey": "api_calls", "quantity": 500, "meterType": "prepaid",
			"balanceDebited": 500, "remainingBalance": 9500, "billable": true,
			"createdAt": "2026-09-29T12:00:00Z",
		}, true
	case "UsageOperation":
		return map[string]any{"id": "0199aad1-f00d-7ae0-935f-cde0bd9f3ba5", "customerId": "customer_123", "meterKey": "api_calls", "quantity": 500, "meterType": "prepaid", "status": "accepted", "denialReason": nil, "balanceDebited": 500, "remainingBalance": 9500, "billable": true, "replayCount": 1, "lastReplayedAt": "2026-09-29T12:01:00Z", "createdAt": "2026-09-29T12:00:00Z"}, true
	case "UsageOperations":
		operation, _ := consumptionExample("UsageOperation")
		return map[string]any{"operations": []any{operation}, "nextCursor": nil}, true
	default:
		return nil, false
	}
}
