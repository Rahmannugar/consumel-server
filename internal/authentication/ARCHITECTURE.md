# Authentication Architecture

Authlier owns reusable authentication behavior and its standard `net/http`
routes. Consumel owns user resolution, tenant access, HTTP policy, and product
workflows.

PostgreSQL is authoritative for accounts and sessions. Redis holds verification
challenges, distributed abuse controls, and short-lived session cache entries.

An authenticated request resolves the opaque session, maps the Authlier subject
to a Consumel user, and loads active organization access. Internal administrator
access remains separate.

Consumel retains at most three active sessions per account. Password and
verification attempts use distributed limits, and automated authentication
email is sent through Resend from the configured no-reply address.
