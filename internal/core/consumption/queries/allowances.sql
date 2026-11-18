-- name: AllowanceByBalance :one
SELECT * FROM recurring_allowances WHERE balance_id=$1;

-- name: DatabaseTime :one
SELECT consumption_entitlement_time()::timestamptz;

-- name: CaptureEntitlementTime :one
SELECT set_config('consumel.entitlement_at', COALESCE(NULLIF(current_setting('consumel.entitlement_at', true), ''), clock_timestamp()::text), true)::text;

-- name: CreateAllowance :one
INSERT INTO recurring_allowances(id,balance_id,anchor_at,next_period_at,version)
VALUES($1,$2,$3,$3,1) RETURNING *;

-- name: IncrementAllowanceVersion :one
UPDATE recurring_allowances SET version=version+1 WHERE id=$1 AND version=$2 RETURNING *;

-- name: InsertAllowanceRevision :exec
INSERT INTO allowance_revisions(allowance_id,version,effective_at,quantity,rollover_cap,enabled,source_type,source_id)
VALUES($1,$2,$3,$4,$5,$6,$7,$8);

-- name: EffectiveAllowanceRevision :one
SELECT * FROM allowance_revisions WHERE allowance_id=$1 AND effective_at<=$2
ORDER BY effective_at DESC,version DESC LIMIT 1;

-- name: PendingAllowanceRevision :one
SELECT * FROM allowance_revisions WHERE allowance_id=$1 AND effective_at>$2
ORDER BY effective_at,version DESC LIMIT 1;

-- name: LastAllowancePeriod :one
SELECT p.*, COALESCE(g.remaining_quantity,0)::bigint AS remaining_quantity
FROM allowance_periods p LEFT JOIN entitlement_grants g ON g.id=p.grant_id
WHERE p.allowance_id=$1 ORDER BY p.period_start DESC LIMIT 1;

-- name: InsertAllowanceGrant :one
INSERT INTO entitlement_grants(id,balance_id,source_type,granted_quantity,remaining_quantity,expires_at)
VALUES($1,$2,'recurring_allowance',$3,$3,$4) RETURNING *;

-- name: InsertAllowancePeriod :exec
INSERT INTO allowance_periods(allowance_id,period_start,period_end,revision,quantity,carried_quantity,skipped_periods,grant_id)
VALUES($1,$2,$3,$4,$5,$6,$7,$8);

-- name: AdvanceAllowance :exec
UPDATE recurring_allowances SET next_period_at=$2,retry_at=now(),retry_attempts=0 WHERE id=$1;

-- name: DueAllowanceBalances :many
SELECT a.balance_id,a.retry_attempts FROM recurring_allowances a
JOIN balances b ON b.id=a.balance_id
JOIN project_environment_meters m ON m.project_environment_id=b.project_environment_id AND m.meter_id=b.meter_id AND m.archived_at IS NULL
JOIN project_environments e ON e.id=b.project_environment_id
JOIN projects p ON p.id=e.project_id
JOIN organizations o ON o.id=p.organization_id AND o.deleted_at IS NULL AND o.suspended_at IS NULL
WHERE a.next_period_at<=now() AND a.retry_at<=now()
ORDER BY a.next_period_at,a.id LIMIT $1;

-- name: DelayAllowanceRetry :exec
UPDATE recurring_allowances SET retry_at=now()+interval '1 minute',retry_attempts=LEAST(retry_attempts,2147483646)+1 WHERE balance_id=$1;

-- name: CustomerAllowanceBalances :many
SELECT b.id FROM balances b JOIN customers c ON c.id=b.customer_id
JOIN recurring_allowances a ON a.balance_id=b.id
JOIN project_environment_meters m ON m.project_environment_id=b.project_environment_id AND m.meter_id=b.meter_id AND m.archived_at IS NULL
WHERE b.project_environment_id=$1 AND c.customer_id=$2 AND a.next_period_at<=now()
ORDER BY b.id;

-- name: ActiveAllowanceBalanceForUpdate :one
SELECT b.* FROM balances b
JOIN project_environment_meters m ON m.project_environment_id=b.project_environment_id AND m.meter_id=b.meter_id AND m.archived_at IS NULL
JOIN project_environments e ON e.id=b.project_environment_id
JOIN projects p ON p.id=e.project_id
JOIN organizations o ON o.id=p.organization_id AND o.deleted_at IS NULL AND o.suspended_at IS NULL
WHERE b.id=$1 FOR UPDATE OF b;

-- name: ListAllowancePeriods :many
SELECT p.*,r.source_type,r.source_id
FROM allowance_periods p
JOIN allowance_revisions r ON r.allowance_id=p.allowance_id AND r.version=p.revision
WHERE p.allowance_id=sqlc.arg(allowance_id)
 AND (sqlc.narg(before_start)::timestamptz IS NULL OR p.period_start<sqlc.narg(before_start)::timestamptz)
ORDER BY p.period_start DESC LIMIT sqlc.arg(page_size);

-- name: DueSubjectAllowanceBalance :one
SELECT b.id FROM balances b JOIN customers c ON c.id=b.customer_id
JOIN recurring_allowances a ON a.balance_id=b.id
JOIN meters m ON m.id=b.meter_id
JOIN project_environment_meters pem ON pem.project_environment_id=b.project_environment_id AND pem.meter_id=b.meter_id AND pem.archived_at IS NULL
WHERE b.project_environment_id=$1 AND c.customer_id=$2 AND m.meter_key=$3 AND a.next_period_at<=now();
