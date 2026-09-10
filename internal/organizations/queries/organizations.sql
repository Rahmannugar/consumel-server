-- name: CreateOrganization :one
INSERT INTO organizations (id, owner_user_id, name)
VALUES ($1, $2, $3)
RETURNING id, owner_user_id, name, created_at, updated_at, deleted_at, suspended_at;

-- name: CreateOrganizationMembership :one
INSERT INTO organization_memberships (organization_id, user_id, role, status)
VALUES ($1, $2, $3, $4)
RETURNING organization_id, user_id, role, status, created_at, updated_at, removed_at;

-- name: GetOrganizationMembership :one
SELECT organization_id, user_id, role, status, created_at, updated_at, removed_at
FROM organization_memberships
WHERE organization_id = $1 AND user_id = $2;
