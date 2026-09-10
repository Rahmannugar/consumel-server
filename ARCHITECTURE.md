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
