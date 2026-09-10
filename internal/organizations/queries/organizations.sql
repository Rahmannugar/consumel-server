-- name: CreateOrganization :one
INSERT INTO organizations (id, name)
VALUES ($1, $2)
RETURNING id, name, created_at, updated_at;

-- name: CreateOrganizationMembership :one
INSERT INTO organization_memberships (organization_id, user_id, role, status)
VALUES ($1, $2, $3, $4)
RETURNING organization_id, user_id, role, status, created_at, updated_at;

-- name: GetOrganizationMembership :one
SELECT organization_id, user_id, role, status, created_at, updated_at
FROM organization_memberships
WHERE organization_id = $1 AND user_id = $2;
