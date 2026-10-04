-- name: ListConsumptionOperations :many
SELECT *
FROM consumption_operations
WHERE project_environment_id = sqlc.arg(project_environment_id)
  AND status <> 'pending'
  AND created_at >= sqlc.arg(from_time)
  AND created_at < sqlc.arg(to_time)
  AND (
      sqlc.arg(status_filter)::text = ''
      OR status = sqlc.arg(status_filter)::text
  )
  AND (
      sqlc.arg(customer_filter)::text = ''
      OR request_customer_id = sqlc.arg(customer_filter)::text
  )
  AND (
      sqlc.arg(meter_filter)::text = ''
      OR request_meter_key = sqlc.arg(meter_filter)::text
  )
  AND (
      sqlc.narg(cursor_created_at)::timestamptz IS NULL
      OR (created_at, id) < (
          sqlc.narg(cursor_created_at)::timestamptz,
          sqlc.narg(cursor_id)::uuid
      )
  )
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(page_size);

-- name: ConsumptionOperationByID :one
SELECT *
FROM consumption_operations
WHERE project_environment_id = sqlc.arg(project_environment_id)
  AND id = sqlc.arg(id)
  AND status <> 'pending';

-- name: AggregateConsumptionOperations :many
SELECT
    date_trunc(sqlc.arg(bucket_interval)::text, created_at)::timestamptz AS bucket_start,
    COUNT(*) FILTER (WHERE status = 'accepted')::bigint AS accepted_operations,
    COUNT(*) FILTER (WHERE status = 'denied')::bigint AS denied_operations,
    COALESCE(SUM(requested_quantity) FILTER (WHERE status = 'accepted'), 0)::bigint AS accepted_quantity,
    COALESCE(SUM(requested_quantity) FILTER (WHERE status = 'denied'), 0)::bigint AS denied_quantity,
    COUNT(*) FILTER (WHERE billable)::bigint AS billable_operations
FROM consumption_operations
WHERE project_environment_id = sqlc.arg(project_environment_id)
  AND status <> 'pending'
  AND created_at >= sqlc.arg(from_time)
  AND created_at < sqlc.arg(to_time)
  AND (
      sqlc.arg(customer_filter)::text = ''
      OR request_customer_id = sqlc.arg(customer_filter)::text
  )
  AND (
      sqlc.arg(meter_filter)::text = ''
      OR request_meter_key = sqlc.arg(meter_filter)::text
  )
GROUP BY bucket_start
ORDER BY bucket_start;
