-- name: CreateProject :one
INSERT INTO projects (id, organization_id, name, slug)
VALUES ($1, $2, $3, $4)
ON CONFLICT (organization_id, slug) DO NOTHING
RETURNING id, organization_id, name, slug, created_at, updated_at;

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
