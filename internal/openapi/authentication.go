package openapi

import (
	"time"
)

// @Summary Return the signed-in account and active organization access.
// @Tags Authentication
// @Success 200 {object} Account "Completed successfully."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 429 {object} RateLimited "Too many attempts were made."
// @Failure 500 {object} ServerError "The request could not be completed."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /account [get]
func GetAccount() {}

// @Summary Unlink Google when another sign-in method remains.
// @Tags Authentication
// @Success 204 "Completed without a response body."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 429 {object} RateLimited "Too many attempts were made."
// @Failure 500 {object} ServerError "The request could not be completed."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /account/google [delete]
func DeleteAccountGoogle() {}

// @Summary Start linking Google to the signed-in account.
// @Tags Authentication
// @Success 200 {object} AuthorizationURL "Completed successfully."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 429 {object} RateLimited "Too many attempts were made."
// @Failure 500 {object} ServerError "The request could not be completed."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /account/google [post]
func PostAccountGoogle() {}

// @Summary Change the account password.
// @Tags Authentication
// @Param body body ChangePasswordRequest true "Current and replacement passwords."
// @Success 200 {object} User "Completed successfully."
// @Failure 400 {object} BadRequest "The JSON body or one of its fields is invalid."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 429 {object} RateLimited "Too many attempts were made."
// @Failure 500 {object} ServerError "The request could not be completed."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /auth/change-password [post]
func PostAuthChangePassword() {}

// @Summary Queue a single-use password-reset link when the account exists.
// @Tags Authentication
// @Param body body EmailRequest true "Email address."
// @Success 202 "The request was accepted without revealing whether the account exists. An eligible account's reset email was queued for asynchronous delivery."
// @Failure 400 {object} BadRequest "The JSON body or one of its fields is invalid."
// @Failure 429 {object} RateLimited "Too many attempts were made."
// @Failure 500 {object} PasswordResetFailed "The reset workflow could not be completed."
// @Router /auth/forgot-password [post]
func PostAuthForgotPassword() {}

// @Summary Start Google sign-in.
// @Tags Authentication
// @Success 200 {object} AuthorizationURL "Completed successfully."
// @Failure 429 {object} RateLimited "Too many attempts were made."
// @Failure 500 {object} ServerError "The request could not be completed."
// @Router /auth/google [post]
func PostAuthGoogle() {}

// @Summary Complete Google sign-in and redirect to the client.
// @Tags Authentication
// @Success 303 "Redirects to the client after authentication."
// @Failure 400 {object} BadRequest "The JSON body or one of its fields is invalid."
// @Failure 429 {object} RateLimited "Too many attempts were made."
// @Failure 500 {object} ServerError "The request could not be completed."
// @Router /auth/google/callback [get]
func GetAuthGoogleCallback() {}

// @Summary Return the account's active sessions.
// @Tags Authentication
// @Success 200 {object} Sessions "Completed successfully."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 429 {object} RateLimited "Too many attempts were made."
// @Failure 500 {object} ServerError "The request could not be completed."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /auth/list-sessions [get]
func GetAuthListSessions() {}

// @Summary Remove password sign-in when another method remains.
// @Tags Authentication
// @Param body body RemovePasswordRequest true "Current password."
// @Success 204 "Completed without a response body."
// @Failure 400 {object} BadRequest "The JSON body or one of its fields is invalid."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 429 {object} RateLimited "Too many attempts were made."
// @Failure 500 {object} ServerError "The request could not be completed."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /auth/remove-password [post]
func PostAuthRemovePassword() {}

// @Summary Queue a new verification code when the account is eligible.
// @Tags Authentication
// @Param body body EmailRequest true "Email address."
// @Success 202 "The request was accepted without revealing account state. When eligible, a new verification code was queued for asynchronous delivery."
// @Failure 400 {object} BadRequest "The JSON body or one of its fields is invalid."
// @Failure 429 {object} RateLimited "Too many attempts were made."
// @Failure 500 {object} EmailVerificationFailed "The verification email could not be queued."
// @Router /auth/resend-verification [post]
func PostAuthResendVerification() {}

// @Summary Replace the password with a valid reset token.
// @Tags Authentication
// @Param body body ResetPasswordRequest true "Reset token and replacement password."
// @Success 200 {object} User "The password was replaced and existing sessions were revoked."
// @Failure 400 {object} ResetPasswordInvalid "The request, reset token, or replacement password is invalid."
// @Failure 429 {object} RateLimited "Too many attempts were made."
// @Failure 500 {object} PasswordResetFailed "The reset workflow could not be completed."
// @Router /auth/reset-password [post]
func PostAuthResetPassword() {}

// @Summary Revoke every account session except the current session.
// @Tags Authentication
// @Success 200 {object} Session "Completed successfully."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 429 {object} RateLimited "Too many attempts were made."
// @Failure 500 {object} ServerError "The request could not be completed."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /auth/revoke-other-sessions [post]
func PostAuthRevokeOtherSessions() {}

// @Summary Revoke one session belonging to the account.
// @Tags Authentication
// @Param body body RevokeSessionRequest true "Session to revoke."
// @Success 204 "Completed without a response body."
// @Failure 400 {object} BadRequest "The JSON body or one of its fields is invalid."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 429 {object} RateLimited "Too many attempts were made."
// @Failure 500 {object} ServerError "The request could not be completed."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /auth/revoke-session [post]
func PostAuthRevokeSession() {}

// @Summary Revoke every session belonging to the account.
// @Tags Authentication
// @Success 204 "Completed without a response body."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 429 {object} RateLimited "Too many attempts were made."
// @Failure 500 {object} ServerError "The request could not be completed."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /auth/revoke-sessions [post]
func PostAuthRevokeSessions() {}

// @Summary Return the current Authlier session.
// @Tags Authentication
// @Success 200 {object} Session "Completed successfully."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 429 {object} RateLimited "Too many attempts were made."
// @Failure 500 {object} ServerError "The request could not be completed."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /auth/session [get]
func GetAuthSession() {}

// @Summary Add password sign-in to the account.
// @Tags Authentication
// @Param body body SetPasswordRequest true "Password to add."
// @Success 200 {object} User "Completed successfully."
// @Failure 400 {object} BadRequest "The JSON body or one of its fields is invalid."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 429 {object} RateLimited "Too many attempts were made."
// @Failure 500 {object} ServerError "The request could not be completed."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /auth/set-password [post]
func PostAuthSetPassword() {}

// @Summary Sign in with email and password.
// @Tags Authentication
// @Param body body CredentialsRequest true "Email and password credentials."
// @Success 200 {object} UserSession "The credentials were accepted and a browser session cookie was issued."
// @Failure 400 {object} BadRequest "The JSON body or one of its fields is invalid."
// @Failure 401 {object} InvalidCredentials "The email address or password is incorrect."
// @Failure 403 {object} EmailNotVerified "The credentials are valid, but email verification is required. A fresh code was queued when the account was eligible."
// @Failure 429 {object} RateLimited "Too many attempts were made."
// @Failure 500 {object} SignInFailed "Authentication or session creation could not be completed."
// @Router /auth/sign-in [post]
func PostAuthSignIn() {}

// @Summary Revoke the current session.
// @Tags Authentication
// @Success 204 "Completed without a response body."
// @Failure 401 {object} NotAuthenticated "Authentication is required."
// @Failure 429 {object} RateLimited "Too many attempts were made."
// @Failure 500 {object} ServerError "The request could not be completed."
// @Security productionCookieSession
// @Security localCookieSession
// @Router /auth/sign-out [post]
func PostAuthSignOut() {}

// @Summary Create an account and queue its verification code.
// @Tags Authentication
// @Param body body CredentialsRequest true "Email and password credentials."
// @Success 201 {object} User "The unverified account was created and its verification code was queued for asynchronous delivery. No session is created until verification succeeds."
// @Failure 400 {object} BadRequest "The JSON body or one of its fields is invalid."
// @Failure 409 {object} RegistrationUnavailable "Registration cannot be completed for this email address. The response does not disclose existing account state."
// @Failure 429 {object} RateLimited "Too many attempts were made."
// @Failure 500 {object} SignUpFailed "Account creation or verification delivery could not be completed."
// @Router /auth/sign-up [post]
func PostAuthSignUp() {}

// @Summary Verify the email code and sign in.
// @Tags Authentication
// @Param body body VerifyEmailRequest true "Email address and verification code."
// @Success 200 {object} UserSession "The email was verified and a browser session cookie was issued."
// @Failure 400 {object} EmailVerificationInvalid "The request body or verification code is invalid."
// @Failure 429 {object} RateLimited "Too many attempts were made."
// @Failure 500 {object} VerifyEmailFailed "Verification or session creation could not be completed."
// @Router /auth/verify-email [post]
func PostAuthVerifyEmail() {}

type Account struct {
	Organizations []OrganizationAccess `json:"organizations" validate:"required,max=1"`
	Session       AccountSession       `json:"session" validate:"required"`
	User          AccountUser          `json:"user" validate:"required"`
}

type AccountSession struct {
	CreatedAt time.Time `json:"createdAt" validate:"required" example:"2026-09-25T12:00:00Z" format:"date-time"`
	ExpiresAt time.Time `json:"expiresAt" validate:"required" example:"2026-10-02T12:00:00Z" format:"date-time"`
	ID        string    `json:"id" validate:"required" example:"01K5A7Q72E9WPJ8D4J13FQ0A6R"`
}

type AccountUser struct {
	Email string `json:"email" validate:"required" example:"developer@example.com" format:"email"`
	ID    string `json:"id" validate:"required" example:"01K5A7PZ9SA93YN3PX84B9G6KB"`
}

type AuthorizationURL struct {
	URL string `json:"url" validate:"required" example:"https://accounts.google.com/o/oauth2/v2/auth?..." format:"uri"`
}

type ChangePasswordRequest struct {
	CurrentPassword     string `json:"currentPassword" validate:"required" example:"Previous horse 6!"`
	NewPassword         string `json:"newPassword" validate:"required,max=128,min=8" example:"Correct horse 7!" format:"password" pattern:"^(?=.*[A-Z])(?=.*[0-9])(?=.*[^\\p{L}\\p{N}\\s]).{8,128}$"`
	RevokeOtherSessions bool   `json:"revokeOtherSessions" example:"true"`
}

type CredentialsRequest struct {
	Email    string `json:"email" validate:"required" example:"developer@example.com" format:"email"`
	Password string `json:"password" validate:"required,max=128,min=8" example:"Correct horse 7!" format:"password" pattern:"^(?=.*[A-Z])(?=.*[0-9])(?=.*[^\\p{L}\\p{N}\\s]).{8,128}$"`
}

type EmailRequest struct {
	Email string `json:"email" validate:"required" example:"developer@example.com" format:"email"`
}

type ListedSession struct {
	CreatedAt time.Time `json:"createdAt" validate:"required" example:"2026-09-25T12:00:00Z" format:"date-time"`
	Current   bool      `json:"current" validate:"required" example:"true"`
	ExpiresAt time.Time `json:"expiresAt" validate:"required" example:"2026-10-02T12:00:00Z" format:"date-time"`
	ID        string    `json:"id" validate:"required" example:"01K5A7Q72E9WPJ8D4J13FQ0A6R"`
	SubjectID string    `json:"subjectId" validate:"required" example:"01K5A7PZ9SA93YN3PX84B9G6KB"`
}

type OrganizationAccess struct {
	ID            string  `json:"id" validate:"required" example:"01K5A80AZ99MGRM9Q7K0SZV8XJ"`
	Name          string  `json:"name" validate:"required" example:"Acme"`
	Owner         bool    `json:"owner" validate:"required" example:"true"`
	RoleID        string  `json:"roleId" validate:"required" example:"01K5A80JPQ1PVX1XBQXF8VZC52"`
	RoleName      string  `json:"roleName" validate:"required" example:"Admin"`
	RoleSystemKey *string `json:"roleSystemKey" example:"admin"`
}

type RemovePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" validate:"required" example:"Correct horse 7!"`
}

type ResetPasswordRequest struct {
	NewPassword string `json:"newPassword" validate:"required,max=128,min=8" example:"Correct horse 7!" format:"password" pattern:"^(?=.*[A-Z])(?=.*[0-9])(?=.*[^\\p{L}\\p{N}\\s]).{8,128}$"`
	Token       string `json:"token" validate:"required" example:"single-use-reset-token"`
}

type RevokeSessionRequest struct {
	SessionID string `json:"sessionId" validate:"required" example:"01K5A7Q72E9WPJ8D4J13FQ0A6R"`
}

type Session struct {
	Session SessionDetails `json:"session" validate:"required"`
}

type SessionDetails struct {
	CreatedAt time.Time `json:"createdAt" validate:"required" example:"2026-09-25T12:00:00Z" format:"date-time"`
	ExpiresAt time.Time `json:"expiresAt" validate:"required" example:"2026-10-02T12:00:00Z" format:"date-time"`
	ID        string    `json:"id" validate:"required" example:"01K5A7Q72E9WPJ8D4J13FQ0A6R"`
	SubjectID string    `json:"subjectId" validate:"required" example:"01K5A7PZ9SA93YN3PX84B9G6KB"`
}

type Sessions struct {
	Sessions []ListedSession `json:"sessions" validate:"required"`
}

type SetPasswordRequest struct {
	Password string `json:"password" validate:"required,max=128,min=8" example:"Correct horse 7!" format:"password" pattern:"^(?=.*[A-Z])(?=.*[0-9])(?=.*[^\\p{L}\\p{N}\\s]).{8,128}$"`
}

type User struct {
	User UserDetails `json:"user" validate:"required"`
}

type UserDetails struct {
	Email string `json:"email" validate:"required" example:"developer@example.com" format:"email"`
	ID    string `json:"id" validate:"required" example:"01K5A7PZ9SA93YN3PX84B9G6KB"`
}

type UserSession struct {
	Session SessionDetails `json:"session" validate:"required"`
	User    UserDetails    `json:"user" validate:"required"`
}

type VerifyEmailRequest struct {
	Code  string `json:"code" validate:"required" example:"482193" pattern:"^[0-9]{6}$"`
	Email string `json:"email" validate:"required" example:"developer@example.com" format:"email"`
}

type EmailNotVerified struct {
	Error EmailNotVerifiedError `json:"error" validate:"required"`
}

type EmailNotVerifiedError struct {
	Code    string `json:"code" validate:"required" example:"email_not_verified"`
	Message string `json:"message,omitempty"`
}

type EmailVerificationFailed struct {
	Error EmailVerificationFailedError `json:"error" validate:"required"`
}

type EmailVerificationFailedError struct {
	Code    string `json:"code" validate:"required" example:"email_verification_failed"`
	Message string `json:"message,omitempty"`
}

// @description Possible codes: invalid_request, invalid_token.
type EmailVerificationInvalid struct {
	Error EmailVerificationInvalidError `json:"error" validate:"required"`
}

type EmailVerificationInvalidError struct {
	Code    string `json:"code" validate:"required" example:"invalid_request"`
	Message string `json:"message,omitempty"`
}

type InvalidCredentials struct {
	Error InvalidCredentialsError `json:"error" validate:"required"`
}

type InvalidCredentialsError struct {
	Code    string `json:"code" validate:"required" example:"invalid_credentials"`
	Message string `json:"message,omitempty"`
}

// @description Possible codes: password_reset_failed, session_revocation_failed.
type PasswordResetFailed struct {
	Error PasswordResetFailedError `json:"error" validate:"required"`
}

type PasswordResetFailedError struct {
	Code    string `json:"code" validate:"required" example:"password_reset_failed"`
	Message string `json:"message,omitempty"`
}

type RegistrationUnavailable struct {
	Error RegistrationUnavailableError `json:"error" validate:"required"`
}

type RegistrationUnavailableError struct {
	Code    string `json:"code" validate:"required" example:"registration_unavailable"`
	Message string `json:"message,omitempty"`
}

// @description Possible codes: invalid_password, invalid_request, invalid_token.
type ResetPasswordInvalid struct {
	Error ResetPasswordInvalidError `json:"error" validate:"required"`
}

type ResetPasswordInvalidError struct {
	Code    string `json:"code" validate:"required" example:"invalid_password"`
	Message string `json:"message,omitempty"`
}

// @description Possible codes: authentication_failed, email_verification_failed, session_failed.
type SignInFailed struct {
	Error SignInFailedError `json:"error" validate:"required"`
}

type SignInFailedError struct {
	Code    string `json:"code" validate:"required" example:"authentication_failed"`
	Message string `json:"message,omitempty"`
}

// @description Possible codes: authentication_failed, email_verification_failed.
type SignUpFailed struct {
	Error SignUpFailedError `json:"error" validate:"required"`
}

type SignUpFailedError struct {
	Code    string `json:"code" validate:"required" example:"authentication_failed"`
	Message string `json:"message,omitempty"`
}

// @description Possible codes: email_verification_failed, session_failed.
type VerifyEmailFailed struct {
	Error VerifyEmailFailedError `json:"error" validate:"required"`
}

type VerifyEmailFailedError struct {
	Code    string `json:"code" validate:"required" example:"email_verification_failed"`
	Message string `json:"message,omitempty"`
}
