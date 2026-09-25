# Authentication Architecture

Authlier owns reusable authentication behavior and its standard `net/http`
routes. Consumel owns user resolution, tenant access, HTTP policy, and product
workflows.

PostgreSQL is authoritative for accounts and sessions. Redis holds verification
challenges, distributed abuse controls, and short-lived session cache entries.

After resolving the opaque session, an established account loads its Consumel
user and active organization access through one PostgreSQL query keyed by the
Authlier subject ID. First access creates the local user projection and repeats
that account-context read. Internal administrator access remains separate.

Consumel retains at most three active sessions per account. Password and
verification attempts use distributed limits, and automated authentication
email is sent through Resend from the configured no-reply address.
