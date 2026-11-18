# Consumel Server

Consumel is infrastructure for usage-based billing. It helps SaaS companies
meter usage, manage customer balances and entitlements, apply pricing rules,
and connect usage data to the payment providers they already use.

This repository contains the Go HTTP API and workers for Consumel. Implemented
workflows include Authlier authentication, organization and project onboarding,
Sandbox/Live API keys, customers, meters, durable grant balances, atomic usage
consumption, usage analytics, and customer recurring allowances. PostgreSQL owns
durable state; Redis Streams carries asynchronous events, and workers deliver
email and renew allowances. Organization billing, pricing, provider connectors,
and the remaining V1 surfaces follow in their connected implementation slices.

## Technology

- Go 1.26
- Gin
- Koanf
- PostgreSQL with pgx
- Redis
- Authlier v0.5.0
- Resend
- OpenTelemetry
- Tern and sqlc
- Task

## Local Development

Create your local configuration:

```bash
cp .env.example .env
```

Update the PostgreSQL, Redis, Authlier, and Resend settings in `.env`. Generate
`CONSUMEL_AUTH_OTP_HMAC_SECRET` as at least 32 random bytes encoded with base64;
never commit that value.

`CONSUMEL_AUTH_TRUSTED_PROXIES` is a comma-separated list of proxy CIDRs or IP
addresses whose forwarded client-IP headers may be trusted. Leave it empty for
direct local traffic. Production should name only the actual Cloudflare or
reverse-proxy hops; an empty value does not mean every proxy is trusted.

`CONSUMEL_RESEND_NOREPLY_FROM` sends automated authentication mail.
`CONSUMEL_RESEND_HELLO_FROM` is the separately configured sender for future
conversational and marketing mail. Secrets and provider credentials remain in
the local or deployment environment, never in `.env.example`.

Start the complete local environment with Docker:

```bash
task up-build
```

This starts PostgreSQL and Redis with persistent local volumes, applies pending
Tern migrations in a temporary container that is removed after completion,
then starts the local OpenTelemetry Collector and API. Follow the API and
Collector logs with `task logs`, and stop the environment without deleting its
data with `task down`.

After the image exists, use `task up` for normal starts. Use `task up-build`
again after changing the Dockerfile.

To run the API directly on the host instead, start PostgreSQL, Redis, and an
OTLP/HTTP Collector, then apply migrations and start the process:

```bash
task migrate
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 task run-api
```

The API is available at [http://localhost:8080](http://localhost:8080) by
default. The Scalar API reference is available at
[http://localhost:8080/docs](http://localhost:8080/docs), and the
generated OpenAPI document is served at `/openapi.json`.

Koanf loads `.env` first and applies process environment variables as
overrides. `CONSUMEL_ENVIRONMENT` accepts `development` or `production` and
defaults to `development`. Production configuration is supplied by the
deployment environment.

The root `Dockerfile` and `compose.yaml` are local-development tooling.
Production container definitions belong under `deploy/` and are added only
when the production deployment slice begins.

## API

The current endpoint map is documented in [API.md](API.md).

## Validation

Run generation, formatting, tests, lint, and build checks:

```bash
task check
```

`task check` requires the `golangci-lint` version pinned in
`.golangci-lint-version`. Run `task lint` to execute that check independently.

Run the PostgreSQL integration tests with Docker available:

```bash
task test-integration
```

After changing SQL queries or migrations, regenerate the type-safe database
code:

```bash
task generate
```

## License

Licensed under the [Apache License 2.0](LICENSE).
