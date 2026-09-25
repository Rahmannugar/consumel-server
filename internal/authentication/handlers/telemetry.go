package handlers

import (
	"net/http"

	"github.com/Rahmannugar/consumel-server/internal/infra/telemetry"
	"github.com/gin-gonic/gin"
)

type requestTelemetry struct {
	operation   string
	completions telemetry.CompletionDetails
}

var authenticationRequestTelemetry = map[string]requestTelemetry{
	http.MethodGet + " /account": requestDetails(
		"user.account.load",
		completion("user.account.loaded", "User account loaded"),
		completion("user.sign_in.required", "User needs to sign in"),
		completion("user.account.load.failed", "Could not load user account"),
	),
	http.MethodPost + " /auth/sign-up": requestDetails(
		"user.sign_up",
		completion("user.signed_up", "User signed up"),
		completion("user.sign_up.rejected", "Sign-up request rejected"),
		completion("user.sign_up.failed", "Could not sign up user"),
	),
	http.MethodPost + " /auth/sign-in": requestDetails(
		"user.sign_in",
		completion("user.signed_in", "User signed in"),
		completion("user.sign_in.rejected", "Sign-in attempt rejected"),
		completion("user.sign_in.failed", "Could not sign in user"),
	),
	http.MethodPost + " /auth/sign-out": requestDetails(
		"user.sign_out",
		completion("user.signed_out", "User signed out"),
		completion("user.sign_out.rejected", "Sign-out request rejected"),
		completion("user.sign_out.failed", "Could not sign out user"),
	),
	http.MethodGet + " /auth/session": requestDetails(
		"user.session.load",
		completion("user.session.loaded", "User session loaded"),
		completion("user.session.missing", "No active user session"),
		completion("user.session.load.failed", "Could not load user session"),
	),
	http.MethodGet + " /auth/list-sessions": requestDetails(
		"user.sessions.list",
		completion("user.sessions.loaded", "User sessions loaded"),
		completion("user.sign_in.required", "User needs to sign in"),
		completion("user.sessions.list.failed", "Could not load user sessions"),
	),
	http.MethodPost + " /auth/revoke-session": requestDetails(
		"user.session.revoke",
		completion("user.session.revoked", "User session revoked"),
		completion("user.session.revoke.rejected", "Session revoke request rejected"),
		completion("user.session.revoke.failed", "Could not revoke user session"),
	),
	http.MethodPost + " /auth/revoke-other-sessions": requestDetails(
		"user.sessions.revoke_others",
		completion("user.sessions.others_revoked", "Other user sessions revoked"),
		completion("user.sessions.revoke_others.rejected", "Session revoke request rejected"),
		completion("user.sessions.revoke_others.failed", "Could not revoke other user sessions"),
	),
	http.MethodPost + " /auth/revoke-sessions": requestDetails(
		"user.sessions.revoke_all",
		completion("user.sessions.revoked", "User sessions revoked"),
		completion("user.sessions.revoke_all.rejected", "Session revoke request rejected"),
		completion("user.sessions.revoke_all.failed", "Could not revoke user sessions"),
	),
	http.MethodPost + " /auth/change-password": requestDetails(
		"user.password.change",
		completion("user.password.changed", "User password changed"),
		completion("user.password.change.rejected", "Password change rejected"),
		completion("user.password.change.failed", "Could not change user password"),
	),
	http.MethodPost + " /auth/set-password": requestDetails(
		"user.password.set",
		completion("user.password.set", "User password set"),
		completion("user.password.set.rejected", "Password setup rejected"),
		completion("user.password.set.failed", "Could not set user password"),
	),
	http.MethodPost + " /auth/remove-password": requestDetails(
		"user.password.remove",
		completion("user.password.removed", "User password removed"),
		completion("user.password.remove.rejected", "Password removal rejected"),
		completion("user.password.remove.failed", "Could not remove user password"),
	),
	http.MethodPost + " /auth/resend-verification": requestDetails(
		"user.email_verification.send",
		completion("user.email_verification.sent", "Verification email request accepted"),
		completion("user.email_verification.send.rejected", "Verification email request rejected"),
		completion("user.email_verification.send.failed", "Could not handle verification email request"),
	),
	http.MethodPost + " /auth/verify-email": requestDetails(
		"user.email.verify",
		completion("user.email.verified", "User email verified"),
		completion("user.email.verify.rejected", "Email verification rejected"),
		completion("user.email.verify.failed", "Could not verify user email"),
	),
	http.MethodPost + " /auth/forgot-password": requestDetails(
		"user.password_reset.request",
		completion("user.password_reset.requested", "Password reset request accepted"),
		completion("user.password_reset.request.rejected", "Password reset request rejected"),
		completion("user.password_reset.request.failed", "Could not handle password reset request"),
	),
	http.MethodPost + " /auth/reset-password": requestDetails(
		"user.password_reset.complete",
		completion("user.password_reset.completed", "User password reset"),
		completion("user.password_reset.rejected", "Password reset rejected"),
		completion("user.password_reset.failed", "Could not reset user password"),
	),
	http.MethodPost + " /auth/google": requestDetails(
		"user.google_sign_in.start",
		completion("user.google_sign_in.started", "Google sign-in started"),
		completion("user.google_sign_in.start.rejected", "Google sign-in request rejected"),
		completion("user.google_sign_in.start.failed", "Could not start Google sign-in"),
	),
	http.MethodGet + " /auth/google/callback": requestDetails(
		"user.google_sign_in.complete",
		completion("user.google_sign_in.completed", "User signed in with Google"),
		completion("user.google_sign_in.rejected", "Google sign-in rejected"),
		completion("user.google_sign_in.failed", "Could not sign in user with Google"),
	),
	http.MethodPost + " /account/google": requestDetails(
		"user.google_account.link",
		completion("user.google_account.link_started", "Google account linking started"),
		completion("user.google_account.link.rejected", "Google account linking rejected"),
		completion("user.google_account.link.failed", "Could not start Google account linking"),
	),
	http.MethodDelete + " /account/google": requestDetails(
		"user.google_account.unlink",
		completion("user.google_account.unlinked", "Google account unlinked"),
		completion("user.google_account.unlink.rejected", "Google account unlink rejected"),
		completion("user.google_account.unlink.failed", "Could not unlink Google account"),
	),
}

var unknownAuthenticationRequest = requestDetails(
	"user.account.request",
	completion("user.account.request.completed", "Account request completed"),
	completion("user.account.request.rejected", "Account request rejected"),
	completion("user.account.request.failed", "Account request failed"),
)

func RequestTelemetry() gin.HandlerFunc {
	return func(context *gin.Context) {
		key := context.Request.Method + " " + context.Request.URL.Path
		details, exists := authenticationRequestTelemetry[key]
		if !exists {
			details = unknownAuthenticationRequest
		}
		telemetry.SetRequestOperation(
			context.Request.Context(),
			details.operation,
			details.completions,
		)
		context.Next()
	}
}

func requestDetails(
	operation string,
	success telemetry.Completion,
	rejected telemetry.Completion,
	failed telemetry.Completion,
) requestTelemetry {
	return requestTelemetry{
		operation: operation,
		completions: telemetry.CompletionDetails{
			Success:  success,
			Rejected: rejected,
			Failed:   failed,
		},
	}
}

func completion(event string, message string) telemetry.Completion {
	return telemetry.Completion{Event: event, Message: message}
}
