# Health API

Reports whether the API process is alive.
`GET /health/live`

Reports whether the API instance can reach PostgreSQL and is ready to serve
traffic. Returns `503 Service Unavailable` when PostgreSQL cannot be reached.
`GET /health/ready`
