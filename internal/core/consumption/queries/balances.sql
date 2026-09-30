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
    requested_expires_at,
    customer_id,
    meter_id
)
VALUES ($1, $2, $3, 'add', $4, $5, $6, $7, $8, $9)
ON CONFLICT (project_environment_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL
    DO NOTHING
RETURNING *;

-- name: EnsureBalance :one
INSERT INTO balances (id, project_environment_id, customer_id, meter_id, quantity)
VALUES ($1, $2, $3, $4, 0)
ON CONFLICT (project_environment_id, customer_id, meter_id)
DO UPDATE SET customer_id = EXCLUDED.customer_id
RETURNING *;

-- name: BalanceForUpdate :one
SELECT *
FROM balances
WHERE balances.id = $1
FOR UPDATE;

-- name: SpendableEntitlementGrants :many
SELECT *
FROM entitlement_grants
WHERE balance_id = $1
  AND remaining_quantity > 0
  AND (expires_at IS NULL OR expires_at > now())
ORDER BY expires_at ASC NULLS LAST, created_at, id
FOR UPDATE;

-- name: InsertEntitlementGrant :one
INSERT INTO entitlement_grants (
    id,
    balance_id,
    source_type,
    granted_quantity,
    remaining_quantity,
    expires_at,
    created_by_balance_operation_id
)
VALUES ($1, $2, 'manual', $3, $3, $4, $5)
RETURNING *;

-- name: UpdateEntitlementGrantRemaining :exec
UPDATE entitlement_grants
SET remaining_quantity = $2, updated_at = now()
WHERE id = $1;

-- name: RecordBalanceGrantAllocation :exec
INSERT INTO balance_operation_grant_allocations (
    balance_operation_id,
    entitlement_grant_id,
    quantity
)
VALUES ($1, $2, $3);

-- name: RefreshBalanceProjection :one
UPDATE balances
SET
    quantity = (
        SELECT COALESCE(SUM(remaining_quantity), 0)::bigint
        FROM entitlement_grants
        WHERE balance_id = balances.id
          AND remaining_quantity > 0
          AND (expires_at IS NULL OR expires_at > now())
    ),
    updated_at = now()
WHERE balances.id = $1
RETURNING *;

-- name: FinalizeBalanceOperation :one
UPDATE balance_operations
SET
    balance_id = $2,
    resulting_quantity = $3,
    resulting_next_expires_at = $4,
    resulting_created_at = $5,
    resulting_updated_at = $6
WHERE id = $1
RETURNING *;

-- name: RecordBalanceSet :one
INSERT INTO balance_operations (
    id,
    project_environment_id,
    operation_type,
    request_customer_id,
    request_meter_key,
    requested_quantity,
    customer_id,
    meter_id
)
VALUES ($1, $2, 'set', $3, $4, $5, $6, $7)
RETURNING *;

-- name: ActiveEntitlementSummary :one
SELECT
    COALESCE(SUM(remaining_quantity), 0)::bigint AS quantity,
    (
        SELECT expiring.expires_at
        FROM entitlement_grants expiring
        WHERE expiring.balance_id = $1
          AND expiring.remaining_quantity > 0
          AND expiring.expires_at > now()
        ORDER BY expiring.expires_at
        LIMIT 1
    ) AS next_expires_at
FROM entitlement_grants
WHERE balance_id = $1
  AND remaining_quantity > 0
  AND (expires_at IS NULL OR expires_at > now());

-- name: BalanceByPublicKeys :one
SELECT
    b.id,
    b.project_environment_id,
    c.customer_id AS public_customer_id,
    m.meter_key,
    entitlement.quantity,
    entitlement.next_expires_at,
    b.created_at,
    b.updated_at
FROM balances b
JOIN customers c
    ON c.project_environment_id = b.project_environment_id
   AND c.id = b.customer_id
JOIN meters m ON m.id = b.meter_id
JOIN project_environment_meters pem
    ON pem.project_environment_id = b.project_environment_id
   AND pem.meter_id = b.meter_id
   AND pem.archived_at IS NULL
LEFT JOIN LATERAL (
    SELECT
        COALESCE(SUM(g.remaining_quantity), 0)::bigint AS quantity,
        (
            SELECT expiring.expires_at
            FROM entitlement_grants expiring
            WHERE expiring.balance_id = b.id
              AND expiring.remaining_quantity > 0
              AND expiring.expires_at > now()
            ORDER BY expiring.expires_at
            LIMIT 1
        ) AS next_expires_at
    FROM entitlement_grants g
    WHERE g.balance_id = b.id
      AND g.remaining_quantity > 0
      AND (g.expires_at IS NULL OR g.expires_at > now())
) entitlement ON true
WHERE b.project_environment_id = $1
  AND c.customer_id = $2
  AND m.meter_key = $3
;

-- name: ListCustomerBalances :many
SELECT
    b.id,
    b.project_environment_id,
    c.customer_id AS public_customer_id,
    m.meter_key,
    entitlement.quantity,
    entitlement.next_expires_at,
    b.created_at,
    b.updated_at
FROM balances b
JOIN customers c
    ON c.project_environment_id = b.project_environment_id
   AND c.id = b.customer_id
JOIN meters m ON m.id = b.meter_id
JOIN project_environment_meters pem
    ON pem.project_environment_id = b.project_environment_id
   AND pem.meter_id = b.meter_id
   AND pem.archived_at IS NULL
LEFT JOIN LATERAL (
    SELECT
        COALESCE(SUM(g.remaining_quantity), 0)::bigint AS quantity,
        (
            SELECT expiring.expires_at
            FROM entitlement_grants expiring
            WHERE expiring.balance_id = b.id
              AND expiring.remaining_quantity > 0
              AND expiring.expires_at > now()
            ORDER BY expiring.expires_at
            LIMIT 1
        ) AS next_expires_at
    FROM entitlement_grants g
    WHERE g.balance_id = b.id
      AND g.remaining_quantity > 0
      AND (g.expires_at IS NULL OR g.expires_at > now())
) entitlement ON true
WHERE b.project_environment_id = $1
  AND c.customer_id = $2
ORDER BY m.meter_key;
