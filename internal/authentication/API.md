# Authentication API

Authlier's standard `net/http` handler owns the email/password and session
routes below. Consumel mounts that handler under `/api/auth` in Gin. JSON
requests require `Content-Type: application/json`. Browser sessions use the
opaque `consumel_session` HttpOnly cookie.

## Signup and verification

### `POST /api/auth/sign-up/email`

Creates an unverified Authlier account and sends a six-digit verification code.
The request body contains `email` and `password`. A successful request returns
`201 Created`; no authenticated session is created before verification.

### `POST /api/auth/send-verification-email`

Replaces the current code for an unverified account. The request body contains
`email`. A successful request returns `202 Accepted`, including when the email
does not identify an eligible account so the response does not disclose account
existence.

### `POST /api/auth/verify-email`

Consumes the current six-digit code. The request body contains `email` and
`code`, for example:

```json
{
  "email": "owner@example.com",
  "code": "482731"
}
```

Success returns `200 OK` with the verified Authlier user and session and sets
the HttpOnly session cookie. An invalid, expired, replaced, or already-used code
returns `400 Bad Request` with `{"error":"invalid_token"}`. Exceeded abuse
limits return `429 Too Many Requests`.

## Password sessions

- `POST /api/auth/sign-in/email` accepts `email` and `password` and creates a
  session only for a verified account.
- `POST /api/auth/sign-out` revokes the current session and clears its cookie.
- `GET /api/auth/session` returns the current Authlier session.
- `GET /api/auth/list-sessions` returns the account's active sessions.
- `POST /api/auth/revoke-session` revokes the session identified by
  `sessionId`.
- `POST /api/auth/revoke-other-sessions` preserves the current session and
  revokes the account's other sessions.
- `POST /api/auth/revoke-sessions` revokes every account session.
- `POST /api/auth/change-password`, `POST /api/auth/set-password`, and
  `POST /api/auth/remove-password` manage the account's password credential
  subject to Authlier's recent-session and remaining-credential rules.

An account may retain three active sessions. Creating a fourth session revokes
the oldest. Authentication routes have contextual distributed limits by IP,
normalized email, and account where those identities are available.

## Consumel tenant context

### `GET /api/auth/context`

Requires the session cookie. It resolves the Authlier subject to a Consumel
user and returns every active organization membership:

```json
{
  "session": {
    "id": "01997fc4-b5d2-7f6b-a5bc-a548d76d742c",
    "createdAt": "2026-09-25T10:30:00Z",
    "expiresAt": "2026-10-02T10:30:00Z"
  },
  "user": {
    "id": "01997fc4-b6a1-7a21-b9a7-9c366f1124d3"
  },
  "organizations": [
    {
      "id": "01997fc5-113b-70bc-af66-773e2335c418",
      "name": "Northstar Labs",
      "owner": true,
      "roleId": "01997fc5-1142-7f29-b05e-29b81cbfc0b6",
      "roleName": "Admin",
      "roleSystemKey": "admin"
    }
  ]
}
```

The organization list excludes inactive memberships, organizations, and roles.
No membership can grant internal-administrator access. Missing or invalid
sessions return a contextual `401 Unauthorized` response; authoritative lookup
failures return a contextual `500 Internal Server Error` response.
