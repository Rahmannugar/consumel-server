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
