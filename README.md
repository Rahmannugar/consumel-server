# Consumel Server

Consumel Server is the Go backend for Consumel. It currently provides the Gin
HTTP process, liveness and readiness endpoints, Koanf-backed environment
configuration, structured process logs, and graceful shutdown.

## Requirements

- Go 1.26 or newer
- Task for repository commands

## Run the API

```shell
task run-api
```

The API listens on port `8080` by default and accepts connections on all network
interfaces.

## Configuration

Configuration is loaded from process environment variables through Koanf:

- `CONSUMEL_ENVIRONMENT`
- `CONSUMEL_HTTP_PORT`

`CONSUMEL_ENVIRONMENT` accepts `development`, `test`, `staging`, or
`production`. No local environment file is required for the defaults.

See `ARCHITECTURE.md` for the current runtime design and `API.md` for the
contract map.

## Validation

```shell
task check
```

## License

Apache License 2.0.
