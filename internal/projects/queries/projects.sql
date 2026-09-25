-- name: CreateProject :one
INSERT INTO projects (id, organization_id, name)
VALUES ($1, $2, $3)
RETURNING id, organization_id, name, created_at, updated_at;

-- name: CreateProjectEnvironment :one
INSERT INTO project_environments (id, project_id, environment, activated_at)
VALUES ($1, $2, $3, $4)
RETURNING id, project_id, environment, activated_at, created_at;

-- name: ListProjectEnvironments :many
SELECT id, project_id, environment, activated_at, created_at
FROM project_environments
WHERE project_id = $1
ORDER BY environment;
