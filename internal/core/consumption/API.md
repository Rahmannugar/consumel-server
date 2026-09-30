## Public integration API

GET /v1/customers/{customerId}/balances
Returns the active meter balances for one customer in the project environment authenticated by the API key.

GET /v1/customers/{customerId}/balances/{meterKey}
Returns one active customer and meter balance from the project environment authenticated by the API key.

POST /v1/balances
Adds an idempotent, optionally expiring quantity to a customer and meter balance in the project environment authenticated by the API key.

PUT /v1/balances/{customerId}/{meterKey}
Sets one customer and meter balance to an exact quantity in the project environment authenticated by the API key.

POST /v1/consume
Atomically accepts usage for an active meter, persists the usage event and outbox record, and applies the meter type's balance behavior in the project environment authenticated by the API key.

## Dashboard application API

GET /v1/projects/{projectId}/environments/{environment}/customers/{customerId}/balances
Returns the active meter balances for one customer from the signed-in project workspace.

GET /v1/projects/{projectId}/environments/{environment}/customers/{customerId}/balances/{meterKey}
Returns one active customer and meter balance from the signed-in project workspace.

POST /v1/projects/{projectId}/environments/{environment}/balances
Adds an idempotent, optionally expiring quantity to a customer and meter balance from the signed-in project workspace.

PUT /v1/projects/{projectId}/environments/{environment}/balances/{customerId}/{meterKey}
Sets one customer and meter balance to an exact quantity from the signed-in project workspace.

GET /v1/projects/{projectId}/environments/{environment}/events
Returns cursor-paginated accepted and denied usage operations for dashboard inspection.

GET /v1/projects/{projectId}/environments/{environment}/events/stream
Streams newly persisted usage operations to the signed-in project workspace with reconnect cursors.
