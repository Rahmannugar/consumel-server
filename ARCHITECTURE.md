# Consumel Server Architecture

Consumel Server runs as a Go HTTP API backed by PostgreSQL and Redis. Gin routes
requests, Koanf loads startup configuration, pgx manages the database pool, and
Authlier v0.4.0 supplies reusable authentication through its standard
`net/http` handler.

## Runtime flow

The process loads and validates configuration, connects to PostgreSQL and
Redis, runs Authlier's versioned PostgreSQL migrations, constructs the Authlier
handler, and then opens its listener. Gin applies the API's cross-cutting HTTP
policy and mounts Authlier without making Authlier depend on Gin. Consumel-owned
handlers resolve the authenticated Authlier subject into tenant application
state.

## Telemetry

`internal/infra/telemetry` owns OpenTelemetry provider, exporter, propagation,
logger, and generic HTTP instrumentation. The application sends traces,
metrics, and logs to an OTLP/HTTP Collector. New Relic remains a Collector
destination rather than an application dependency.

Every non-health request produces one JSON completion log on stdout and the
same record in the OpenTelemetry pipeline. The record contains a stable
operation, a domain-specific event, a plain human-readable message, method,
route template, response status, duration, request ID, trace ID, outcome, and
bounded error category when applicable. The operation names the attempted
action across outcomes; the event names the result that occurred. Middleware
also owns request spans and bounded traffic, latency, error, and active-request
metrics. Routine successful health probes are suppressed; readiness failures
remain correlated and visible.

PostgreSQL, Redis, and outbound HTTP instrumentation is attached at shared
infrastructure construction boundaries. Handlers do not configure telemetry
exporters or repeat generic tracing and timing logic.

## Lifecycle

The API accepts SIGINT and SIGTERM. It stops accepting new requests, gives
in-flight requests a bounded period to finish, closes the Redis client and
PostgreSQL pool, and then exits. Startup, serving, and shutdown failures produce
a non-zero process exit.

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
the stable Authlier subject ID as the unique external identity. Email is not an
identity link. A conflict-safe insert-or-select flow creates the Consumel user
on first authenticated access, including concurrent first requests. The
organizations domain owns organizations and organization memberships.

Consumel internal administrators belong to a separate identity and
authorization boundary. Tenant roles and organization memberships cannot grant
internal administration access.

An organization stores its owner through `owner_user_id`. Organization creation
also creates its built-in Admin and Developer roles and the owner's active Admin
membership in one transaction. Each membership has one organization role, and
organizations may define custom roles.
A project is created with isolated Sandbox and Live environments in one
transaction. Sandbox starts active; Live remains inactive until explicitly
activated.

## Authentication

PostgreSQL is authoritative for Authlier users, password credentials, email
verification state, and sessions. Redis stores short-lived OTP challenges,
distributed authentication rate-limit state, and the shared session cache.
OTP challenges contain no plaintext code and are consumed once. A failure
after challenge consumption requires the user to request a new code rather
than making the old code reusable.

Browser sessions use opaque HttpOnly cookies and last seven days. Redis caches
resolved sessions for no more than one hour with up to ten percent downward
jitter so many entries do not expire together. Identical cache-miss lookups are
coalesced, and PostgreSQL still resolves established sessions when Redis is
unavailable. New OTP operations and sensitive authentication attempts require
Redis because their shared challenge and abuse-control state must remain
consistent across API instances.

Each Authlier subject may have at most three active sessions. Session creation
uses a PostgreSQL transaction and a subject-scoped advisory lock; a fourth
sign-in revokes the oldest session and invalidates its cache entry.

`GET /api/auth/context` maps the Authlier subject to one Consumel user and all
active tenant memberships. The query excludes removed or suspended
memberships, deleted or suspended organizations, and deleted roles. This tenant
context never resolves or grants internal-administrator access.

Authentication email is rendered through the shared bordered Consumel email
layout and sent by Resend from `noreply@consumel.com`. The separately configured
`hello@consumel.com` sender is reserved for conversational and marketing mail.

## Failure behavior

- A Redis session-cache miss or error falls back to PostgreSQL.
- The broad per-IP HTTP limiter fails open during a Redis outage so existing
  sessions remain usable. Sensitive Authlier attempt guards fail closed.
- OTP issuance and verification fail when Redis cannot provide shared,
  single-use challenge state.
- PostgreSQL failure makes the API unready and prevents authoritative identity,
  session, or tenant-context resolution.
