# Consumel Server

Consumel is infrastructure for usage-based billing. It helps SaaS companies
meter usage, manage customer balances and entitlements, apply pricing rules,
and connect usage data to the payment providers they already use.

This repository contains the Go backend for Consumel. The current server
provides the HTTP API foundation, Authlier email/password authentication,
PostgreSQL persistence, Redis-backed authentication coordination, health
checks, environment-backed configuration, structured logging, graceful
shutdown, and the initial user, organization, organization-membership, project,
and environment persistence flows.

## Technology

- Go 1.26
- Gin
- Koanf
- PostgreSQL with pgx
- Redis
- Authlier v0.4.0
- Resend
- Tern and sqlc
- Task

## Local Development

Create your local configuration:

```bash
cp .env.example .env
```

Update the PostgreSQL, Redis, Authlier, and Resend settings in `.env`, apply the
migrations, then start the API. Generate
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

```bash
task migrate
```

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
