package openapi

import (
	"encoding/json"
	"fmt"
	"sort"
)

type operation struct {
	Method      string
	Path        string
	Summary     string
	Request     string
	SuccessCode string
	Success     string
	Protected   bool
}

var operations = []operation{
	{Method: "get", Path: "/account", Summary: "Return the signed-in account and active organization access.", SuccessCode: "200", Success: "Account", Protected: true},
	{Method: "delete", Path: "/account/google", Summary: "Unlink Google when another sign-in method remains.", SuccessCode: "204", Protected: true},
	{Method: "post", Path: "/account/google", Summary: "Start linking Google to the signed-in account.", SuccessCode: "200", Success: "AuthorizationURL", Protected: true},
	{Method: "post", Path: "/auth/change-password", Summary: "Change the account password.", Request: "ChangePasswordRequest", SuccessCode: "200", Success: "User", Protected: true},
	{Method: "post", Path: "/auth/forgot-password", Summary: "Send a single-use password-reset link when the account exists.", Request: "EmailRequest", SuccessCode: "202"},
	{Method: "post", Path: "/auth/google", Summary: "Start Google sign-in.", SuccessCode: "200", Success: "AuthorizationURL"},
	{Method: "get", Path: "/auth/google/callback", Summary: "Complete Google sign-in and redirect to the client.", SuccessCode: "302"},
	{Method: "get", Path: "/auth/list-sessions", Summary: "Return the account's active sessions.", SuccessCode: "200", Success: "Sessions", Protected: true},
	{Method: "post", Path: "/auth/remove-password", Summary: "Remove password sign-in when another method remains.", Request: "RemovePasswordRequest", SuccessCode: "204", Protected: true},
	{Method: "post", Path: "/auth/resend-verification", Summary: "Send a new verification code when the account exists.", Request: "EmailRequest", SuccessCode: "202"},
	{Method: "post", Path: "/auth/reset-password", Summary: "Replace the password with a valid reset token.", Request: "ResetPasswordRequest", SuccessCode: "200", Success: "User"},
	{Method: "post", Path: "/auth/revoke-other-sessions", Summary: "Revoke every account session except the current session.", SuccessCode: "200", Success: "Session", Protected: true},
	{Method: "post", Path: "/auth/revoke-session", Summary: "Revoke one session belonging to the account.", Request: "RevokeSessionRequest", SuccessCode: "204", Protected: true},
	{Method: "post", Path: "/auth/revoke-sessions", Summary: "Revoke every session belonging to the account.", SuccessCode: "204", Protected: true},
	{Method: "get", Path: "/auth/session", Summary: "Return the current Authlier session.", SuccessCode: "200", Success: "Session", Protected: true},
	{Method: "post", Path: "/auth/set-password", Summary: "Add password sign-in to the account.", Request: "SetPasswordRequest", SuccessCode: "200", Success: "User", Protected: true},
	{Method: "post", Path: "/auth/sign-in", Summary: "Sign in with email and password.", Request: "CredentialsRequest", SuccessCode: "200", Success: "UserSession"},
	{Method: "post", Path: "/auth/sign-out", Summary: "Revoke the current session.", SuccessCode: "204", Protected: true},
	{Method: "post", Path: "/auth/sign-up", Summary: "Create an account and send its verification code.", Request: "CredentialsRequest", SuccessCode: "201", Success: "User"},
	{Method: "post", Path: "/auth/verify-email", Summary: "Verify the email code and sign in.", Request: "VerifyEmailRequest", SuccessCode: "200", Success: "UserSession"},
	{Method: "get", Path: "/health/live", Summary: "Report whether the API process is alive.", SuccessCode: "200", Success: "Health"},
	{Method: "get", Path: "/health/ready", Summary: "Report whether PostgreSQL is reachable.", SuccessCode: "200", Success: "Health"},
}

// Document builds the deterministic OpenAPI contract served by the API and
// written to openapi.json for review and stale-output checks.
func Document() ([]byte, error) {
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
		response := map[string]any{"description": successDescription(endpoint.SuccessCode)}
		if endpoint.Success != "" {
			response["content"] = jsonContent(schemaReference(endpoint.Success), exampleFor(endpoint.Success))
		}
		operationDocument := map[string]any{
			"summary":   endpoint.Summary,
			"tags":      []string{tagFor(endpoint.Path)},
			"responses": operationResponses(endpoint, response),
		}
		if endpoint.Request != "" {
			operationDocument["requestBody"] = map[string]any{
				"required": true,
				"content":  jsonContent(schemaReference(endpoint.Request), exampleFor(endpoint.Request)),
			}
		}
		if endpoint.Protected {
			operationDocument["security"] = []map[string][]string{{"cookieSession": {}}}
		}
		path[endpoint.Method] = operationDocument
	}

	document := map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":       "Consumel API",
			"version":     "0.1.0",
			"description": "Consumel usage-based billing infrastructure API.",
		},
		"paths": paths,
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"cookieSession": map[string]any{"type": "apiKey", "in": "cookie", "name": "consumel_session"},
			},
			"schemas":   schemas(),
			"responses": errorResponses(),
		},
	}
	encoded, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode OpenAPI document: %w", err)
	}
	return append(encoded, '\n'), nil
}

func schemas() map[string]any {
	stringProperty := func() map[string]any { return map[string]any{"type": "string"} }
	password := map[string]any{"type": "string", "minLength": 12, "maxLength": 128, "format": "password", "pattern": `^(?=.*[A-Z])(?=.*[0-9])(?=.*[^\p{L}\p{N}\s]).{12,128}$`}
	return map[string]any{
		"Account":               object([]string{"session", "user", "organizations"}, map[string]any{"session": schemaReference("AccountSession"), "user": schemaReference("AccountUser"), "organizations": map[string]any{"type": "array", "items": schemaReference("OrganizationAccess")}}),
		"AccountSession":        object([]string{"id", "createdAt", "expiresAt"}, map[string]any{"id": stringProperty(), "createdAt": map[string]any{"type": "string", "format": "date-time"}, "expiresAt": map[string]any{"type": "string", "format": "date-time"}}),
		"AccountUser":           object([]string{"id"}, map[string]any{"id": stringProperty()}),
		"AuthorizationURL":      object([]string{"url"}, map[string]any{"url": map[string]any{"type": "string", "format": "uri"}}),
		"ChangePasswordRequest": object([]string{"currentPassword", "newPassword"}, map[string]any{"currentPassword": stringProperty(), "newPassword": password, "revokeOtherSessions": map[string]any{"type": "boolean"}}),
		"CredentialsRequest":    object([]string{"email", "password"}, map[string]any{"email": map[string]any{"type": "string", "format": "email"}, "password": password}),
		"EmailRequest":          object([]string{"email"}, map[string]any{"email": map[string]any{"type": "string", "format": "email"}}),
		"Error":                 object([]string{"error"}, map[string]any{"error": object([]string{"code"}, map[string]any{"code": stringProperty()})}),
		"Health":                object([]string{"status"}, map[string]any{"status": map[string]any{"type": "string", "example": "ok"}}),
		"OrganizationAccess":    object([]string{"id", "name", "owner", "roleId", "roleName"}, map[string]any{"id": stringProperty(), "name": stringProperty(), "owner": map[string]any{"type": "boolean"}, "roleId": stringProperty(), "roleName": stringProperty(), "roleSystemKey": map[string]any{"type": []string{"string", "null"}}}),
		"RemovePasswordRequest": object([]string{"currentPassword"}, map[string]any{"currentPassword": stringProperty()}),
		"ResetPasswordRequest":  object([]string{"token", "newPassword"}, map[string]any{"token": stringProperty(), "newPassword": password}),
		"RevokeSessionRequest":  object([]string{"sessionId"}, map[string]any{"sessionId": stringProperty()}),
		"Session":               object([]string{"session"}, map[string]any{"session": schemaReference("SessionDetails")}),
		"SessionDetails":        object([]string{"id", "subjectId", "createdAt", "expiresAt"}, map[string]any{"id": stringProperty(), "subjectId": stringProperty(), "createdAt": map[string]any{"type": "string", "format": "date-time"}, "expiresAt": map[string]any{"type": "string", "format": "date-time"}}),
		"ListedSession":         object([]string{"id", "subjectId", "createdAt", "expiresAt", "current"}, map[string]any{"id": stringProperty(), "subjectId": stringProperty(), "createdAt": map[string]any{"type": "string", "format": "date-time"}, "expiresAt": map[string]any{"type": "string", "format": "date-time"}, "current": map[string]any{"type": "boolean"}}),
		"Sessions":              object([]string{"sessions"}, map[string]any{"sessions": map[string]any{"type": "array", "items": schemaReference("ListedSession")}}),
		"SetPasswordRequest":    object([]string{"password"}, map[string]any{"password": password}),
		"User":                  object([]string{"user"}, map[string]any{"user": schemaReference("UserDetails")}),
		"UserDetails":           object([]string{"id", "email"}, map[string]any{"id": stringProperty(), "email": map[string]any{"type": "string", "format": "email"}}),
		"UserSession":           object([]string{"user", "session"}, map[string]any{"user": schemaReference("UserDetails"), "session": schemaReference("SessionDetails")}),
		"VerifyEmailRequest":    object([]string{"email", "code"}, map[string]any{"email": map[string]any{"type": "string", "format": "email"}, "code": map[string]any{"type": "string", "pattern": `^[0-9]{6}$`, "example": "482193"}}),
	}
}

func errorResponses() map[string]any {
	return map[string]any{
		"BadRequest":       errorResponse("The request is invalid.", "invalid_request"),
		"NotAuthenticated": errorResponse("Authentication is required.", "not_authenticated"),
		"RateLimited":      errorResponse("Too many attempts were made.", "too_many_attempts"),
		"ServerError":      errorResponse("The request could not be completed.", "authentication_failed"),
	}
}

func errorResponse(description, code string) map[string]any {
	return map[string]any{"description": description, "content": jsonContent(schemaReference("Error"), map[string]any{"error": map[string]any{"code": code}})}
}

func operationResponses(endpoint operation, success map[string]any) map[string]any {
	responses := map[string]any{
		endpoint.SuccessCode: success,
		"500":                responseReference("ServerError"),
	}
	if endpoint.Request != "" || endpoint.Path == "/auth/google/callback" {
		responses["400"] = responseReference("BadRequest")
	}
	if endpoint.Protected {
		responses["401"] = responseReference("NotAuthenticated")
	}
	if tagFor(endpoint.Path) == "Authentication" {
		responses["429"] = responseReference("RateLimited")
	}
	if endpoint.Path == "/health/ready" {
		responses["503"] = map[string]any{
			"description": "PostgreSQL is unavailable.",
			"content":     jsonContent(schemaReference("Health"), map[string]any{"status": "unavailable"}),
		}
	}
	return responses
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

func successDescription(status string) string {
	if status == "201" {
		return "Created."
	}
	if status == "202" {
		return "Accepted without revealing whether the account exists."
	}
	if status == "204" {
		return "Completed without a response body."
	}
	if status == "302" {
		return "Redirects to the client after authentication."
	}
	return "Completed successfully."
}

func tagFor(path string) string {
	if len(path) >= 7 && path[:7] == "/health" {
		return "Health"
	}
	return "Authentication"
}

func exampleFor(name string) map[string]any {
	session := map[string]any{"id": "01K5A7Q72E9WPJ8D4J13FQ0A6R", "subjectId": "01K5A7PZ9SA93YN3PX84B9G6KB", "createdAt": "2026-09-25T12:00:00Z", "expiresAt": "2026-10-02T12:00:00Z"}
	user := map[string]any{"id": "01K5A7PZ9SA93YN3PX84B9G6KB", "email": "developer@example.com"}
	switch name {
	case "CredentialsRequest":
		return map[string]any{"email": "developer@example.com", "password": "Correct horse 7!"}
	case "EmailRequest":
		return map[string]any{"email": "developer@example.com"}
	case "VerifyEmailRequest":
		return map[string]any{"email": "developer@example.com", "code": "482193"}
	case "ResetPasswordRequest":
		return map[string]any{"token": "single-use-reset-token", "newPassword": "Correct horse 7!"}
	case "ChangePasswordRequest":
		return map[string]any{"currentPassword": "Previous horse 6!", "newPassword": "Correct horse 7!", "revokeOtherSessions": true}
	case "SetPasswordRequest":
		return map[string]any{"password": "Correct horse 7!"}
	case "RemovePasswordRequest":
		return map[string]any{"currentPassword": "Correct horse 7!"}
	case "RevokeSessionRequest":
		return map[string]any{"sessionId": "01K5A7Q72E9WPJ8D4J13FQ0A6R"}
	case "AuthorizationURL":
		return map[string]any{"url": "https://accounts.google.com/o/oauth2/v2/auth?..."}
	case "Health":
		return map[string]any{"status": "ok"}
	case "Session":
		return map[string]any{"session": session}
	case "Sessions":
		listedSession := map[string]any{}
		for key, value := range session {
			listedSession[key] = value
		}
		listedSession["current"] = true
		return map[string]any{"sessions": []any{listedSession}}
	case "User":
		return map[string]any{"user": user}
	case "UserSession":
		return map[string]any{"user": user, "session": session}
	case "Account":
		accountSession := map[string]any{"id": session["id"], "createdAt": session["createdAt"], "expiresAt": session["expiresAt"]}
		return map[string]any{"user": map[string]any{"id": user["id"]}, "session": accountSession, "organizations": []any{map[string]any{"id": "01K5A80AZ99MGRM9Q7K0SZV8XJ", "name": "Acme", "owner": true, "roleId": "01K5A80JPQ1PVX1XBQXF8VZC52", "roleName": "Owner", "roleSystemKey": nil}}}
	default:
		return map[string]any{}
	}
}
