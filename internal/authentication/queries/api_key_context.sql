-- name: ResolveActiveProjectAPIKey :one
WITH matched AS MATERIALIZED (
    SELECT
        project_api_keys.id AS api_key_id,
        projects.organization_id,
        projects.id AS project_id,
        project_environments.id AS project_environment_id,
        project_environments.environment
    FROM project_api_keys
    JOIN project_environments
        ON project_environments.id = project_api_keys.project_environment_id
    JOIN projects
        ON projects.id = project_environments.project_id
    JOIN organizations
        ON organizations.id = projects.organization_id
    WHERE project_api_keys.key_hash = $1
      AND project_api_keys.revoked_at IS NULL
      AND project_environments.activated_at IS NOT NULL
      AND organizations.deleted_at IS NULL
      AND organizations.suspended_at IS NULL
), touched AS (
    UPDATE project_api_keys
    SET last_used_at = now()
    FROM matched
    WHERE project_api_keys.id = matched.api_key_id
      AND (
          project_api_keys.last_used_at IS NULL
          OR project_api_keys.last_used_at < now() - interval '5 minutes'
      )
    RETURNING project_api_keys.id
)
SELECT
    matched.api_key_id,
    matched.organization_id,
    matched.project_id,
    matched.project_environment_id,
    matched.environment
FROM matched
LEFT JOIN touched ON touched.id = matched.api_key_id;
