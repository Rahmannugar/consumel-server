-- name: BalanceOperationByIdempotencyKey :one
SELECT *
FROM balance_operations
WHERE project_environment_id = $1
  AND idempotency_key = $2;

-- name: BalanceSubjectByPublicKeys :one
SELECT c.id AS customer_id, m.id AS meter_id
FROM customers c
JOIN meters m ON m.project_id = (
    SELECT project_id FROM project_environments WHERE id = c.project_environment_id
)
JOIN project_environment_meters pem
    ON pem.project_environment_id = c.project_environment_id
   AND pem.meter_id = m.id
   AND pem.archived_at IS NULL
WHERE c.project_environment_id = $1
  AND c.customer_id = $2
  AND m.meter_key = $3;

-- name: ConsumptionCustomerByPublicID :one
SELECT id
FROM customers
WHERE project_environment_id = $1
  AND customer_id = $2;

-- name: ClaimBalanceAddition :one
INSERT INTO balance_operations (
    id,
    project_environment_id,
    idempotency_key,
    operation_type,
    request_customer_id,
    request_meter_key,
    requested_quantity,
    customer_id,
    meter_id
)
VALUES ($1, $2, $3, 'add', $4, $5, $6, $7, $8)
ON CONFLICT (project_environment_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL
    DO NOTHING
RETURNING *;

-- name: AddBalance :one
INSERT INTO balances (
    id,
    project_environment_id,
    customer_id,
    meter_id,
    quantity
)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (project_environment_id, customer_id, meter_id)
DO UPDATE SET
    quantity = balances.quantity + EXCLUDED.quantity,
    updated_at = now()
RETURNING *;

-- name: FinalizeBalanceOperation :one
UPDATE balance_operations
SET
    balance_id = $2,
    resulting_quantity = $3,
    resulting_created_at = $4,
    resulting_updated_at = $5
WHERE id = $1
RETURNING *;

-- name: SetBalance :one
INSERT INTO balances (
    id,
    project_environment_id,
    customer_id,
    meter_id,
    quantity
)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (project_environment_id, customer_id, meter_id)
DO UPDATE SET
    quantity = EXCLUDED.quantity,
    updated_at = now()
RETURNING *;

-- name: RecordBalanceSet :exec
INSERT INTO balance_operations (
    id,
    project_environment_id,
    operation_type,
    request_customer_id,
    request_meter_key,
    requested_quantity,
    customer_id,
    meter_id,
    balance_id,
    resulting_quantity,
    resulting_created_at,
    resulting_updated_at
)
VALUES ($1, $2, 'set', $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: BalanceByPublicKeys :one
SELECT
    b.*,
    c.customer_id AS public_customer_id,
    m.meter_key
FROM balances b
JOIN customers c
    ON c.project_environment_id = b.project_environment_id
   AND c.id = b.customer_id
JOIN meters m ON m.id = b.meter_id
JOIN project_environment_meters pem
    ON pem.project_environment_id = b.project_environment_id
   AND pem.meter_id = b.meter_id
   AND pem.archived_at IS NULL
WHERE b.project_environment_id = $1
  AND c.customer_id = $2
  AND m.meter_key = $3;

-- name: ListCustomerBalances :many
SELECT
    b.*,
    c.customer_id AS public_customer_id,
    m.meter_key
FROM balances b
JOIN customers c
    ON c.project_environment_id = b.project_environment_id
   AND c.id = b.customer_id
JOIN meters m ON m.id = b.meter_id
JOIN project_environment_meters pem
    ON pem.project_environment_id = b.project_environment_id
   AND pem.meter_id = b.meter_id
   AND pem.archived_at IS NULL
WHERE b.project_environment_id = $1
  AND c.customer_id = $2
ORDER BY m.meter_key;
