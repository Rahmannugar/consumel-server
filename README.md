# Consumel Server

Consumel is infrastructure for usage-based billing. It helps SaaS companies
meter usage, manage customer balances and entitlements, apply pricing rules,
and connect usage data to the payment providers they already use.

This repository contains the Go backend for Consumel. The current server
provides the HTTP API foundation, PostgreSQL connectivity, health checks,
environment-backed configuration, structured logging, and graceful shutdown.

## Technology

- Go 1.26
- Gin
- Koanf
- PostgreSQL with pgx
- Task

## Local Development

Create your local configuration:

```bash
cp .env.example .env
```

Update `CONSUMEL_DATABASE_URL` in `.env` for your local PostgreSQL instance,
then start the API:

```bash
task run-api
```

The API is available at [http://localhost:8080](http://localhost:8080) by
default.

Koanf loads `.env` first and applies process environment variables as
overrides. `CONSUMEL_ENVIRONMENT` accepts `development` or `production` and
defaults to `development`. Production configuration is supplied by the
deployment environment.

## API

The current endpoint map is documented in [API.md](API.md).

## Validation

Run formatting, tests, vet, and build checks:

```bash
task check
```

## License

Licensed under the [Apache License 2.0](LICENSE).
