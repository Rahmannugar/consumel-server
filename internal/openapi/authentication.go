package openapi

func authenticationOperations() []operation {
	return []operation{
		{Method: "get", Path: "/account", Summary: "Return the signed-in account and active organization access.", SuccessCode: "200", Success: "Account", Protected: true},
		{Method: "delete", Path: "/account/google", Summary: "Unlink Google when another sign-in method remains.", SuccessCode: "204", Protected: true},
		{Method: "post", Path: "/account/google", Summary: "Start linking Google to the signed-in account.", SuccessCode: "200", Success: "AuthorizationURL", Protected: true},
		{Method: "post", Path: "/auth/change-password", Summary: "Change the account password.", Request: "ChangePasswordRequest", SuccessCode: "200", Success: "User", Protected: true},
		{Method: "post", Path: "/auth/forgot-password", Summary: "Queue a single-use password-reset link when the account exists.", Request: "EmailRequest", SuccessCode: "202", SuccessDescription: "The request was accepted without revealing whether the account exists. An eligible account's reset email was queued for asynchronous delivery.", Errors: map[string]string{"400": "BadRequest", "429": "RateLimited", "500": "PasswordResetFailed"}},
		{Method: "post", Path: "/auth/google", Summary: "Start Google sign-in.", SuccessCode: "200", Success: "AuthorizationURL"},
		{Method: "get", Path: "/auth/google/callback", Summary: "Complete Google sign-in and redirect to the client.", SuccessCode: "303"},
		{Method: "get", Path: "/auth/list-sessions", Summary: "Return the account's active sessions.", SuccessCode: "200", Success: "Sessions", Protected: true},
		{Method: "post", Path: "/auth/remove-password", Summary: "Remove password sign-in when another method remains.", Request: "RemovePasswordRequest", SuccessCode: "204", Protected: true},
		{Method: "post", Path: "/auth/resend-verification", Summary: "Queue a new verification code when the account is eligible.", Request: "EmailRequest", SuccessCode: "202", SuccessDescription: "The request was accepted without revealing account state. When eligible, a new verification code was queued for asynchronous delivery.", Errors: map[string]string{"400": "BadRequest", "429": "RateLimited", "500": "EmailVerificationFailed"}},
		{Method: "post", Path: "/auth/reset-password", Summary: "Replace the password with a valid reset token.", Request: "ResetPasswordRequest", SuccessCode: "200", Success: "User", SuccessDescription: "The password was replaced and existing sessions were revoked.", Errors: map[string]string{"400": "ResetPasswordInvalid", "429": "RateLimited", "500": "ResetPasswordFailed"}},
		{Method: "post", Path: "/auth/revoke-other-sessions", Summary: "Revoke every account session except the current session.", SuccessCode: "200", Success: "Session", Protected: true},
		{Method: "post", Path: "/auth/revoke-session", Summary: "Revoke one session belonging to the account.", Request: "RevokeSessionRequest", SuccessCode: "204", Protected: true},
		{Method: "post", Path: "/auth/revoke-sessions", Summary: "Revoke every session belonging to the account.", SuccessCode: "204", Protected: true},
		{Method: "get", Path: "/auth/session", Summary: "Return the current Authlier session.", SuccessCode: "200", Success: "Session", Protected: true},
		{Method: "post", Path: "/auth/set-password", Summary: "Add password sign-in to the account.", Request: "SetPasswordRequest", SuccessCode: "200", Success: "User", Protected: true},
		{Method: "post", Path: "/auth/sign-in", Summary: "Sign in with email and password.", Request: "CredentialsRequest", SuccessCode: "200", Success: "UserSession", SuccessDescription: "The credentials were accepted and a browser session cookie was issued.", Errors: map[string]string{"400": "BadRequest", "401": "InvalidCredentials", "403": "EmailNotVerified", "429": "RateLimited", "500": "SignInFailed"}},
		{Method: "post", Path: "/auth/sign-out", Summary: "Revoke the current session.", SuccessCode: "204", Protected: true},
		{Method: "post", Path: "/auth/sign-up", Summary: "Create an account and queue its verification code.", Request: "CredentialsRequest", SuccessCode: "201", Success: "User", SuccessDescription: "The unverified account was created and its verification code was queued for asynchronous delivery. No session is created until verification succeeds.", Errors: map[string]string{"400": "BadRequest", "409": "RegistrationUnavailable", "429": "RateLimited", "500": "SignUpFailed"}},
		{Method: "post", Path: "/auth/verify-email", Summary: "Verify the email code and sign in.", Request: "VerifyEmailRequest", SuccessCode: "200", Success: "UserSession", SuccessDescription: "The email was verified and a browser session cookie was issued.", Errors: map[string]string{"400": "EmailVerificationInvalid", "429": "RateLimited", "500": "VerifyEmailFailed"}},
	}
}

func authenticationSchemas() map[string]any {
	stringProperty := func() map[string]any { return map[string]any{"type": "string"} }
	password := map[string]any{"type": "string", "minLength": 8, "maxLength": 128, "format": "password", "pattern": `^(?=.*[A-Z])(?=.*[0-9])(?=.*[^\p{L}\p{N}\s]).{8,128}$`}
	return map[string]any{
		"Account":               object([]string{"session", "user", "organizations"}, map[string]any{"session": schemaReference("AccountSession"), "user": schemaReference("AccountUser"), "organizations": map[string]any{"type": "array", "maxItems": 1, "items": schemaReference("OrganizationAccess")}}),
		"AccountSession":        object([]string{"id", "createdAt", "expiresAt"}, map[string]any{"id": stringProperty(), "createdAt": map[string]any{"type": "string", "format": "date-time"}, "expiresAt": map[string]any{"type": "string", "format": "date-time"}}),
		"AccountUser":           object([]string{"id", "email"}, map[string]any{"id": stringProperty(), "email": map[string]any{"type": "string", "format": "email"}}),
		"AuthorizationURL":      object([]string{"url"}, map[string]any{"url": map[string]any{"type": "string", "format": "uri"}}),
		"ChangePasswordRequest": object([]string{"currentPassword", "newPassword"}, map[string]any{"currentPassword": stringProperty(), "newPassword": password, "revokeOtherSessions": map[string]any{"type": "boolean"}}),
		"CredentialsRequest":    object([]string{"email", "password"}, map[string]any{"email": map[string]any{"type": "string", "format": "email"}, "password": password}),
		"EmailRequest":          object([]string{"email"}, map[string]any{"email": map[string]any{"type": "string", "format": "email"}}),
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

func authenticationErrorResponses() map[string]any {
	return map[string]any{
		"BadRequest":               errorResponse("The JSON body or one of its fields is invalid.", "invalid_request"),
		"EmailNotVerified":         errorResponse("The credentials are valid, but email verification is required. A fresh code was queued when the account was eligible.", "email_not_verified"),
		"EmailVerificationFailed":  errorResponse("The verification email could not be queued.", "email_verification_failed"),
		"EmailVerificationInvalid": errorResponseExamples("The request body or verification code is invalid.", "invalid_request", "invalid_token"),
		"InvalidCredentials":       errorResponse("The email address or password is incorrect.", "invalid_credentials"),
		"InvalidAPIKey":            errorResponseWithMessage("The project API key is missing, malformed, revoked, replaced, or inactive.", "invalid_api_key", "Provide an active project environment API key."),
		"NotAuthenticated":         errorResponse("Authentication is required.", "not_authenticated"),
		"PasswordResetFailed":      errorResponseExamples("The reset workflow could not be completed.", "password_reset_failed", "session_revocation_failed"),
		"RateLimited":              errorResponse("Too many attempts were made.", "too_many_attempts"),
		"RegistrationUnavailable":  errorResponse("Registration cannot be completed for this email address. The response does not disclose existing account state.", "registration_unavailable"),
		"ResetPasswordInvalid":     errorResponseExamples("The request, reset token, or replacement password is invalid.", "invalid_request", "invalid_token", "invalid_password"),
		"SignInFailed":             errorResponseExamples("Authentication or session creation could not be completed.", "authentication_failed", "email_verification_failed", "session_failed"),
		"SignUpFailed":             errorResponseExamples("Account creation or verification delivery could not be completed.", "authentication_failed", "email_verification_failed"),
		"VerifyEmailFailed":        errorResponseExamples("Verification or session creation could not be completed.", "email_verification_failed", "session_failed"),
	}
}

func authenticationExample(name string) (map[string]any, bool) {
	session := map[string]any{"id": "01K5A7Q72E9WPJ8D4J13FQ0A6R", "subjectId": "01K5A7PZ9SA93YN3PX84B9G6KB", "createdAt": "2026-09-25T12:00:00Z", "expiresAt": "2026-10-02T12:00:00Z"}
	user := map[string]any{"id": "01K5A7PZ9SA93YN3PX84B9G6KB", "email": "developer@example.com"}
	switch name {
	case "CredentialsRequest":
		return map[string]any{"email": "developer@example.com", "password": "Correct horse 7!"}, true
	case "EmailRequest":
		return map[string]any{"email": "developer@example.com"}, true
	case "VerifyEmailRequest":
		return map[string]any{"email": "developer@example.com", "code": "482193"}, true
	case "ResetPasswordRequest":
		return map[string]any{"token": "single-use-reset-token", "newPassword": "Correct horse 7!"}, true
	case "ChangePasswordRequest":
		return map[string]any{"currentPassword": "Previous horse 6!", "newPassword": "Correct horse 7!", "revokeOtherSessions": true}, true
	case "SetPasswordRequest":
		return map[string]any{"password": "Correct horse 7!"}, true
	case "RemovePasswordRequest":
		return map[string]any{"currentPassword": "Correct horse 7!"}, true
	case "RevokeSessionRequest":
		return map[string]any{"sessionId": session["id"]}, true
	case "AuthorizationURL":
		return map[string]any{"url": "https://accounts.google.com/o/oauth2/v2/auth?..."}, true
	case "Session":
		return map[string]any{"session": session}, true
	case "Sessions":
		listed := map[string]any{}
		for key, value := range session {
			listed[key] = value
		}
		listed["current"] = true
		return map[string]any{"sessions": []any{listed}}, true
	case "User":
		return map[string]any{"user": user}, true
	case "UserSession":
		return map[string]any{"user": user, "session": session}, true
	case "Account":
		accountSession := map[string]any{"id": session["id"], "createdAt": session["createdAt"], "expiresAt": session["expiresAt"]}
		return map[string]any{"user": user, "session": accountSession, "organizations": []any{map[string]any{"id": "01K5A80AZ99MGRM9Q7K0SZV8XJ", "name": "Acme", "owner": true, "roleId": "01K5A80JPQ1PVX1XBQXF8VZC52", "roleName": "Admin", "roleSystemKey": "admin"}}}, true
	default:
		return nil, false
	}
}
