-- name: CreateProject :one
INSERT INTO projects (id, organization_id, name, slug)
VALUES ($1, $2, $3, $4)
ON CONFLICT (organization_id, slug) DO NOTHING
RETURNING id, organization_id, name, slug, created_at, updated_at;

-- name: ProjectNameExists :one
SELECT EXISTS (
    SELECT 1
    FROM projects
    WHERE organization_id = $1
      AND lower(btrim(name)) = lower(btrim(sqlc.arg(project_name)))
);

-- name: CreateProjectEnvironment :one
INSERT INTO project_environments (id, project_id, environment, activated_at)
VALUES ($1, $2, $3, $4)
RETURNING id, project_id, environment, activated_at, created_at;

-- name: ListProjectEnvironments :many
SELECT id, project_id, environment, activated_at, created_at
FROM project_environments
WHERE project_id = $1
ORDER BY CASE environment WHEN 'sandbox' THEN 0 ELSE 1 END;

-- name: ListAccessibleProjects :many
SELECT
    projects.id,
    projects.organization_id,
    projects.name,
    projects.slug,
    projects.created_at,
    projects.updated_at,
    organizations.name AS organization_name,
    project_environments.id AS environment_id,
    project_environments.environment,
    project_environments.activated_at AS environment_activated_at,
    project_environments.created_at AS environment_created_at
FROM organization_memberships
JOIN organizations
    ON organizations.id = organization_memberships.organization_id
JOIN projects
    ON projects.organization_id = organizations.id
JOIN project_environments
    ON project_environments.project_id = projects.id
WHERE organization_memberships.user_id = $1
  AND organization_memberships.status = 'active'
  AND organization_memberships.removed_at IS NULL
  AND organizations.deleted_at IS NULL
  AND organizations.suspended_at IS NULL
ORDER BY
    projects.created_at,
    projects.id,
    CASE project_environments.environment WHEN 'sandbox' THEN 0 ELSE 1 END;

-- name: ListProjectPortfolio :many
SELECT
    projects.id,
    projects.name,
    projects.slug,
    project_environments.id AS environment_id,
    project_environments.activated_at,
    COUNT(consumption_operations.id) FILTER (
        WHERE consumption_operations.status = 'accepted'
    )::bigint AS allowed_operations,
    COUNT(consumption_operations.id) FILTER (
        WHERE consumption_operations.status = 'denied'
    )::bigint AS blocked_operations,
    MAX(consumption_operations.created_at)::timestamptz AS last_activity_at
FROM organization_memberships
JOIN organizations
    ON organizations.id = organization_memberships.organization_id
JOIN projects
    ON projects.organization_id = organizations.id
JOIN project_environments
    ON project_environments.project_id = projects.id
   AND project_environments.environment = sqlc.arg(environment)
LEFT JOIN consumption_operations
    ON consumption_operations.project_environment_id = project_environments.id
   AND consumption_operations.status <> 'pending'
   AND consumption_operations.created_at >= sqlc.arg(from_time)
   AND consumption_operations.created_at < sqlc.arg(to_time)
WHERE organization_memberships.user_id = sqlc.arg(user_id)
  AND organization_memberships.status = 'active'
  AND organization_memberships.removed_at IS NULL
  AND organizations.deleted_at IS NULL
  AND organizations.suspended_at IS NULL
GROUP BY
    projects.id,
    projects.name,
    projects.slug,
    project_environments.id,
    project_environments.activated_at
ORDER BY projects.created_at, projects.id;

-- name: AggregateProjectPortfolio :many
SELECT
    date_trunc(sqlc.arg(bucket_interval)::text, consumption_operations.created_at)::timestamptz AS bucket_start,
    COUNT(*) FILTER (WHERE consumption_operations.status = 'accepted')::bigint AS allowed_operations,
    COUNT(*) FILTER (WHERE consumption_operations.status = 'denied')::bigint AS blocked_operations
FROM organization_memberships
JOIN organizations
    ON organizations.id = organization_memberships.organization_id
JOIN projects
    ON projects.organization_id = organizations.id
JOIN project_environments
    ON project_environments.project_id = projects.id
   AND project_environments.environment = sqlc.arg(environment)
JOIN consumption_operations
    ON consumption_operations.project_environment_id = project_environments.id
WHERE organization_memberships.user_id = sqlc.arg(user_id)
  AND organization_memberships.status = 'active'
  AND organization_memberships.removed_at IS NULL
  AND organizations.deleted_at IS NULL
  AND organizations.suspended_at IS NULL
  AND consumption_operations.status <> 'pending'
  AND consumption_operations.created_at >= sqlc.arg(from_time)
  AND consumption_operations.created_at < sqlc.arg(to_time)
GROUP BY bucket_start
ORDER BY bucket_start;

-- name: AccessibleProjectEnvironment :one
SELECT
    project_environments.id,
    project_environments.project_id,
    project_environments.environment,
    project_environments.activated_at,
    project_environments.created_at
FROM project_environments
JOIN projects ON projects.id = project_environments.project_id
JOIN organizations ON organizations.id = projects.organization_id
JOIN organization_memberships
    ON organization_memberships.organization_id = organizations.id
WHERE projects.id = $1
  AND project_environments.environment = $2
  AND organization_memberships.user_id = $3
  AND organization_memberships.status = 'active'
  AND organization_memberships.removed_at IS NULL
  AND organizations.deleted_at IS NULL
  AND organizations.suspended_at IS NULL;

-- name: LockAccessibleProjectEnvironment :one
SELECT
    project_environments.id,
    project_environments.project_id,
    project_environments.environment,
    project_environments.activated_at,
    project_environments.created_at
FROM project_environments
JOIN projects ON projects.id = project_environments.project_id
JOIN organizations ON organizations.id = projects.organization_id
JOIN organization_memberships
    ON organization_memberships.organization_id = organizations.id
WHERE projects.id = $1
  AND project_environments.environment = $2
  AND organization_memberships.user_id = $3
  AND organization_memberships.status = 'active'
  AND organization_memberships.removed_at IS NULL
  AND organizations.deleted_at IS NULL
  AND organizations.suspended_at IS NULL
FOR UPDATE OF project_environments;

-- name: ActiveProjectAPIKey :one
SELECT
    project_api_keys.id,
    project_api_keys.project_environment_id,
    project_api_keys.key_prefix,
    project_api_keys.last_four,
    project_api_keys.created_at,
    project_api_keys.last_used_at,
    project_api_keys.revoked_at
FROM project_api_keys
WHERE project_api_keys.project_environment_id = $1
  AND project_api_keys.revoked_at IS NULL;

-- name: ActivateAccessibleProjectEnvironment :one
UPDATE project_environments
SET activated_at = COALESCE(project_environments.activated_at, now())
FROM projects, organizations, organization_memberships
WHERE project_environments.project_id = projects.id
  AND projects.organization_id = organizations.id
  AND organization_memberships.organization_id = organizations.id
  AND projects.id = $1
  AND project_environments.environment = $2
  AND organization_memberships.user_id = $3
  AND organization_memberships.status = 'active'
  AND organization_memberships.removed_at IS NULL
  AND organizations.deleted_at IS NULL
  AND organizations.suspended_at IS NULL
RETURNING
    project_environments.id,
    project_environments.project_id,
    project_environments.environment,
    project_environments.activated_at,
    project_environments.created_at;

-- name: CreateProjectAPIKey :one
INSERT INTO project_api_keys (
    id,
    project_environment_id,
    key_hash,
    key_prefix,
    last_four,
    created_by_user_id
)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING
    id,
    project_environment_id,
    key_prefix,
    last_four,
    created_at,
    last_used_at,
    revoked_at;

-- name: RevokeActiveProjectAPIKey :one
UPDATE project_api_keys
SET revoked_at = now(), revoked_by_user_id = $2
WHERE project_environment_id = $1
  AND revoked_at IS NULL
RETURNING
    id,
    project_environment_id,
    key_prefix,
    last_four,
    created_at,
    last_used_at,
    revoked_at;
