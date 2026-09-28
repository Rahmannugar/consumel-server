-- name: CreateCustomer :one
INSERT INTO customers (
    id,
    project_environment_id,
    customer_id,
    name,
    email,
    metadata_plan,
    metadata_country,
    metadata_location
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: CustomerByPublicID :one
SELECT *
FROM customers
WHERE project_environment_id = $1
  AND customer_id = $2;

-- name: ListCustomers :many
SELECT *
FROM customers
WHERE project_environment_id = sqlc.arg(project_environment_id)
  AND (
      sqlc.narg(cursor_created_at)::timestamptz IS NULL
      OR (created_at, id) < (
          sqlc.narg(cursor_created_at)::timestamptz,
          sqlc.narg(cursor_id)::uuid
      )
  )
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(page_size);

-- name: UpdateCustomer :one
UPDATE customers
SET
    name = $3,
    email = $4,
    metadata_plan = $5,
    metadata_country = $6,
    metadata_location = $7,
    updated_at = now()
WHERE project_environment_id = $1
  AND customer_id = $2
RETURNING *;
