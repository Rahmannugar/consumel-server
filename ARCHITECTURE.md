# Consumel Server Architecture

Consumel is a Go modular monolith with separate HTTP API and worker processes.
Gin owns HTTP routing, Koanf loads configuration, and published Authlier supplies
reusable authentication through its standard `net/http` handler.

## Domain boundaries

Domains own their services, persistence interfaces, queries, HTTP contracts,
and consequential tests. Handlers translate HTTP into domain operations;
PostgreSQL, Redis, and external clients implement domain-owned interfaces.

- [Authentication](internal/authentication/ARCHITECTURE.md) resolves Authlier identities into Consumel users and current tenant access.
- [Organizations](internal/organizations/ARCHITECTURE.md) owns organizations, ownership, memberships, and roles.
- [Onboarding](internal/onboarding/ARCHITECTURE.md) creates the first organization and project transactionally.
- [Projects](internal/projects/ARCHITECTURE.md) owns projects, Sandbox/Live environments, integration keys, and portfolio reads.
- [Customers](internal/customers/ARCHITECTURE.md) owns environment-scoped customer records.
- [Meters](internal/meters/ARCHITECTURE.md) owns stable meter definitions and environment activation.
- [Consumption](internal/core/consumption/ARCHITECTURE.md) owns grants, balances, recurring customer allowances, atomic usage decisions, and their history and analytics.

Tenant access never grants Consumel internal-administrator access. The internal
administration identity and authorization boundary is separate.

## Durable state and asynchronous work

PostgreSQL is authoritative. Tern owns ordered migrations; domains own sqlc
queries and pgx repositories. Authlier owns its versioned authentication schema.
Normal reads exclude lifecycle-inactive resources. Database constraints and
short, consistently ordered locks protect transactional invariants.

Business writes and their outbox events commit together. The relay publishes
committed events to Redis Streams; consumers process at least once with durable
idempotency and explicit retry/recovery behavior. Slow providers stay outside
synchronous product transactions. Each workload has its own bounded capacity.

Prepared statements remain cached, with parameter-specific PostgreSQL planning
through `plan_cache_mode=force_custom_plan` to preserve selective query plans.
Application pools bound client connections; deployment uses PgBouncer to bound
server-side connections independently of API instance count.

## Authentication and isolation

Authlier authenticates opaque browser sessions. PostgreSQL owns session expiry
and revocation; Redis caches sessions and coordinates single-use OTPs and abuse
controls. Session-cache failures fall back to PostgreSQL, while sensitive flows
requiring shared challenge or rate-limit state fail closed.

Dashboard requests resolve the authenticated subject's current organization
access before entering a domain. Public integration requests resolve an active
hashed API key into organization, project, and environment context. Sandbox
and Live remain isolated, and inactive organizations cannot resolve access.

## Telemetry and process lifecycle

`internal/infra/telemetry` owns OpenTelemetry setup, correlation, and generic
HTTP/job instrumentation. Structured logs remain on stdout; OTLP exports to a
Collector, keeping New Relic replaceable. Dependencies are instrumented at their
construction boundaries. Successful health probes are suppressed. SSE records
setup and delivery failures without tracing entire connection lifetimes.

Liveness reports process availability; readiness checks PostgreSQL. Startup
validates configuration and connects dependencies before serving. Shutdown stops
new work, bounds in-flight completion, and closes dependencies before telemetry.
Request/header limits and workload-specific deadlines bound resource use.

The [API index](API.md) links endpoint domains. OpenAPI owns schemas and
responses; domain architecture documents own their durable rules. Investigation
history, benchmark runs, and slice walkthroughs belong in the working history
and conversation deep dives.
