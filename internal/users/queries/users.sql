-- name: CreateUser :one
INSERT INTO users (id, clerk_user_id)
VALUES ($1, $2)
RETURNING id, clerk_user_id, created_at;

-- name: GetUserByClerkID :one
SELECT id, clerk_user_id, created_at
FROM users
WHERE clerk_user_id = $1;
