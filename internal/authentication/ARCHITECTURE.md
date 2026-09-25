# Authentication Architecture

The authentication domain connects Authlier's reusable identity behavior to
Consumel's tenant model. Authlier remains an external `net/http` component;
Gin owns server routing and cross-cutting HTTP policy, while Consumel services
own user and organization resolution.

PostgreSQL is the durable authority for accounts, password credentials, email
verification status, and sessions. Redis holds only short-lived operational
state: OTP challenges, distributed abuse controls, and cached session records.
The OTP code is never stored as plaintext. The configured HMAC secret binds the
six-digit code to the normalized email, and Redis consumes the proof once.

An authenticated request resolves in this order:

1. Authlier validates the opaque cookie against Redis or PostgreSQL.
2. The users domain resolves the stable Authlier subject to one internal user.
3. The organizations domain loads every active tenant membership and role.
4. The handler returns tenant context without consulting the separate
   internal-administrator identity domain.

Session creation is serialized per subject in PostgreSQL. At most three active
sessions are retained, and a fourth sign-in revokes the oldest. Redis cache
entries last at most one hour with downward jitter; PostgreSQL remains the
authority for the seven-day session lifetime and revocation state.

Authentication attempts use distributed Redis limits keyed by HMAC-protected
IP, normalized email, or account ID as appropriate. The limiter combines a
small immediate allowance with a hard rolling-window ceiling. The general
per-IP middleware permits requests when Redis fails so existing sessions can
reach PostgreSQL, while sensitive password and OTP guards reject attempts when
shared abuse-control state is unavailable.

Verification mail uses a shared HTML layout with a bordered card and a plain
text alternative. Resend sends automated authentication messages through the
configured no-reply sender.
