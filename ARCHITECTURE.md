# Consumel Server Architecture

Consumel Server currently runs as a Go HTTP API backed by PostgreSQL. Gin routes
requests, Koanf loads startup configuration, and pgx manages the database
connection pool.

## Runtime flow

The process loads and validates configuration, connects to PostgreSQL, and then
opens its listener. Requests pass through Gin to the domain that owns the
endpoint. Redis and external providers are not connected yet.

## Lifecycle

The API accepts SIGINT and SIGTERM. It stops accepting new requests, gives
in-flight requests a bounded period to finish, closes its PostgreSQL pool, and
then exits. Startup, serving, and shutdown failures produce a non-zero process
exit.

The server limits the time allowed to receive request headers and request
bodies, and it closes idle keep-alive connections after a bounded interval.
There is no global response-write timeout. Response deadlines belong to the
endpoint behavior that requires them.

## Health

Liveness reports that the process can answer HTTP requests. Readiness checks
that PostgreSQL is reachable before reporting the API instance as ready.

## Persistence

PostgreSQL is the durable source of truth. Tern owns ordered schema migrations,
and sqlc generates pgx-backed query code from domain-owned SQL.

Consumel uses internal UUIDv7 identifiers for users, organizations, projects,
and project environments. The users domain owns tenant-user records and stores
Clerk's identifier separately as the unique external identity. The
organizations domain owns organizations and organization memberships.

Consumel internal administrators belong to a separate identity and
authorization boundary. Tenant roles and organization memberships cannot grant
internal administration access.

An organization is created with its active owner membership in one transaction.
A project is created with isolated Sandbox and Live environments in one
transaction. Sandbox starts active; Live remains inactive until explicitly
activated.
