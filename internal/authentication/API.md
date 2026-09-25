# Authentication API

Swagger/OpenAPI owns request schemas, responses, examples, and error details.

### `POST /api/auth/sign-up`
Creates an email-and-password account and automatically sends its verification code.

### `POST /api/auth/resend-verification`
Sends a new verification code without revealing whether the account exists.

### `POST /api/auth/verify-email`
Verifies the email code and signs in the account.

### `POST /api/auth/sign-in`
Signs in a verified account with email and password.

### `POST /api/auth/sign-out`
Revokes the current session and clears its cookie.

### `GET /api/auth/session`
Returns the current Authlier session.

### `GET /api/auth/list-sessions`
Returns the account's active sessions.

### `POST /api/auth/revoke-session`
Revokes one session belonging to the account.

### `POST /api/auth/revoke-other-sessions`
Revokes every account session except the current session.

### `POST /api/auth/revoke-sessions`
Revokes every session belonging to the account.

### `POST /api/auth/change-password`
Changes the account password after verifying the current password.

### `POST /api/auth/set-password`
Adds password sign-in to an account that uses another sign-in method.

### `POST /api/auth/remove-password`
Removes password sign-in when another sign-in method remains.

### `POST /api/auth/forgot-password`
Sends a single-use password-reset link without revealing whether the account exists.

### `POST /api/auth/reset-password`
Replaces the password with a valid reset token and revokes existing sessions.

### `POST /api/auth/google`
Starts Google sign-in and returns the provider authorization URL.

### `GET /api/auth/google/callback`
Completes Google sign-in and redirects the browser to the client application.

### `GET /api/account`
Returns the signed-in Consumel user, session, and active organization access.

### `POST /api/account/google`
Starts linking one Google identity to the signed-in account.

### `DELETE /api/account/google`
Unlinks the Google identity when another sign-in method remains.
