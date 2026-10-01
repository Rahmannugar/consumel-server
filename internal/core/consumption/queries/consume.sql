-- name: ConsumptionOperationByIdempotencyKey :one
SELECT *
FROM consumption_operations
WHERE project_environment_id = $1
  AND idempotency_key = $2;

-- name: RecordConsumptionReplay :one
UPDATE consumption_operations
SET replay_count = replay_count + 1, last_replayed_at = now()
WHERE id = $1
RETURNING *;

-- name: ActiveConsumptionMeter :one
SELECT
    m.id,
    m.meter_type,
    pe.environment AS environment_name
FROM project_environment_meters pem
JOIN meters m ON m.id = pem.meter_id
JOIN project_environments pe ON pe.id = pem.project_environment_id
WHERE pem.project_environment_id = $1
  AND pem.archived_at IS NULL
  AND m.meter_key = $2;

-- name: EnsureConsumptionCustomer :one
INSERT INTO customers (id, project_environment_id, customer_id)
VALUES ($1, $2, $3)
ON CONFLICT (project_environment_id, customer_id)
DO UPDATE SET customer_id = EXCLUDED.customer_id
RETURNING id;

-- name: ClaimConsumptionOperation :one
INSERT INTO consumption_operations (
    id,
    project_environment_id,
    idempotency_key,
    request_customer_id,
    request_meter_key,
    requested_quantity,
    customer_id,
    meter_id,
    meter_type,
    source_api_key_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (project_environment_id, idempotency_key)
DO NOTHING
RETURNING *;

-- name: BalanceForConsumption :one
SELECT *
FROM balances
WHERE project_environment_id = $1
  AND customer_id = $2
  AND meter_id = $3
FOR UPDATE;

-- name: EnsureHybridBalance :exec
INSERT INTO balances (id, project_environment_id, customer_id, meter_id, quantity)
VALUES ($1, $2, $3, $4, 0)
ON CONFLICT (project_environment_id, customer_id, meter_id)
DO NOTHING;

-- name: RecordConsumptionGrantAllocation :exec
INSERT INTO consumption_grant_allocations (
    consumption_operation_id,
    entitlement_grant_id,
    quantity
)
VALUES ($1, $2, $3);

-- name: AcceptConsumptionOperation :one
UPDATE consumption_operations
SET
    status = 'accepted',
    balance_id = $2,
    balance_debited = $3,
    resulting_balance = $4,
    billable = $5
WHERE id = $1
RETURNING *;

-- name: DenyConsumptionOperation :one
UPDATE consumption_operations
SET
    status = 'denied',
    denial_reason = 'insufficient_balance',
    balance_id = $2,
    resulting_balance = $3
WHERE id = $1
RETURNING *;
