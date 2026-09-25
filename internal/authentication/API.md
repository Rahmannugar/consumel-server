# Authentication API

Swagger/OpenAPI owns request schemas, responses, examples, and error details.

### `GET /account`
Returns the signed-in Consumel user, session, and active organization access.

### `DELETE /account/google`
Unlinks the Google identity when another sign-in method remains.

### `POST /account/google`
Starts linking one Google identity to the signed-in account.

### `POST /auth/change-password`
Changes the account password after verifying the current password.

### `POST /auth/forgot-password`
Sends a single-use password-reset link without revealing whether the account exists.

### `POST /auth/google`
Starts Google sign-in and returns the provider authorization URL.

### `GET /auth/google/callback`
Completes Google sign-in and redirects the browser to the client application.

### `GET /auth/list-sessions`
Returns the account's active sessions.

### `POST /auth/remove-password`
Removes password sign-in when another sign-in method remains.

### `POST /auth/resend-verification`
Sends a new verification code without revealing whether the account exists.

### `POST /auth/reset-password`
Replaces the password with a valid reset token and revokes existing sessions.

### `POST /auth/revoke-other-sessions`
Revokes every account session except the current session.

### `POST /auth/revoke-session`
Revokes one session belonging to the account.

### `POST /auth/revoke-sessions`
Revokes every session belonging to the account.

### `GET /auth/session`
Returns the current Authlier session.

### `POST /auth/set-password`
Adds password sign-in to an account that uses another sign-in method.

### `POST /auth/sign-in`
Signs in a verified account with email and password.

### `POST /auth/sign-out`
Revokes the current session and clears its cookie.

### `POST /auth/sign-up`
Creates an email-and-password account and automatically sends its verification code.

### `POST /auth/verify-email`
Verifies the email code and signs in the account.
