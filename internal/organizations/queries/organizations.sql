-- name: CreateOrganization :one
INSERT INTO organizations (id, owner_user_id, name)
VALUES ($1, $2, $3)
RETURNING id, owner_user_id, name, created_at, updated_at, deleted_at, suspended_at;

-- name: CreateOrganizationMembership :one
INSERT INTO organization_memberships (organization_id, user_id, role_id, status)
VALUES ($1, $2, $3, $4)
RETURNING organization_id, user_id, role_id, status, created_at, updated_at, removed_at;

-- name: CreateOrganizationRole :one
INSERT INTO organization_roles (id, organization_id, name, system_key)
VALUES ($1, $2, $3, $4)
RETURNING id, organization_id, name, system_key, created_at, updated_at, deleted_at;

-- name: GetOrganizationMembership :one
SELECT organization_id, user_id, role_id, status, created_at, updated_at, removed_at
FROM organization_memberships
WHERE organization_id = $1 AND user_id = $2;

-- name: ListActiveOrganizationAccessByUser :many
SELECT
    organizations.id AS organization_id,
    organizations.name AS organization_name,
    organizations.owner_user_id = organization_memberships.user_id AS owner,
    organization_roles.id AS role_id,
    organization_roles.name AS role_name,
    organization_roles.system_key AS role_system_key
FROM organization_memberships
JOIN organizations
    ON organizations.id = organization_memberships.organization_id
JOIN organization_roles
    ON organization_roles.organization_id = organization_memberships.organization_id
   AND organization_roles.id = organization_memberships.role_id
WHERE organization_memberships.user_id = $1
  AND organization_memberships.status = 'active'
  AND organization_memberships.removed_at IS NULL
  AND organizations.deleted_at IS NULL
  AND organizations.suspended_at IS NULL
  AND organization_roles.deleted_at IS NULL
ORDER BY organizations.created_at, organizations.id;

-- name: ListOrganizationRoles :many
SELECT id, organization_id, name, system_key, created_at, updated_at, deleted_at
FROM organization_roles
WHERE organization_id = $1 AND deleted_at IS NULL
ORDER BY name;
