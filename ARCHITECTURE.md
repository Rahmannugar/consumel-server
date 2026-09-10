# Consumel Server Architecture

Consumel Server currently runs as a stateless Go HTTP API. Gin routes requests,
and Koanf loads startup configuration from process environment variables.

## Runtime flow

The process loads and validates configuration before opening its listener.
Requests pass through Gin to the domain that owns the endpoint. The current
foundation exposes only health behavior and does not connect to PostgreSQL,
Redis, or external providers.

## Lifecycle

The API accepts SIGINT and SIGTERM. It stops accepting new requests, gives
in-flight requests a bounded period to finish, and then exits. Startup, serving,
and shutdown failures produce a non-zero process exit.

The server limits the time allowed to receive request headers and request
bodies, and it closes idle keep-alive connections after a bounded interval.
There is no global response-write timeout. Response deadlines belong to the
endpoint behavior that requires them.

## Health

Liveness reports that the process can answer HTTP requests. Readiness reports
that the dependencies required by the current runtime are ready. Because this
foundation has no external runtime dependencies, both checks currently return
healthy when the HTTP process can serve them.
