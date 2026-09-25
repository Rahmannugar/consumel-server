-- name: GetAccountContextByAuthlierSubjectID :many
SELECT
    users.id AS user_id,
    users.authlier_subject_id,
    users.created_at AS user_created_at,
    organizations.id AS organization_id,
    organizations.name AS organization_name,
    COALESCE(organizations.owner_user_id = users.id, false)::boolean AS owner,
    organization_roles.id AS role_id,
    organization_roles.name AS role_name,
    organization_roles.system_key AS role_system_key
FROM users
LEFT JOIN organization_memberships
    ON organization_memberships.user_id = users.id
   AND organization_memberships.status = 'active'
   AND organization_memberships.removed_at IS NULL
LEFT JOIN organizations
    ON organizations.id = organization_memberships.organization_id
   AND organizations.deleted_at IS NULL
   AND organizations.suspended_at IS NULL
LEFT JOIN organization_roles
    ON organization_roles.organization_id = organizations.id
   AND organization_roles.id = organization_memberships.role_id
   AND organization_roles.deleted_at IS NULL
WHERE users.authlier_subject_id = $1
ORDER BY organizations.created_at NULLS LAST, organizations.id;
