ALTER TABLE organizations
    ADD COLUMN owner_user_id uuid;

UPDATE organizations
SET owner_user_id = organization_memberships.user_id
FROM organization_memberships
WHERE organization_memberships.organization_id = organizations.id
  AND organization_memberships.role = 'owner';

ALTER TABLE organizations
    ALTER COLUMN owner_user_id SET NOT NULL,
    ADD CONSTRAINT organizations_owner_user_id_fkey
        FOREIGN KEY (owner_user_id) REFERENCES users (id);

ALTER TABLE organization_memberships
    DROP CONSTRAINT organization_memberships_role_valid;

UPDATE organization_memberships
SET role = 'admin'
WHERE role = 'owner';

ALTER TABLE organization_memberships
    ADD CONSTRAINT organization_memberships_role_valid CHECK (
        role IN ('admin', 'developer')
    ),
    ADD COLUMN removed_at timestamptz;

ALTER TABLE organizations
    ADD COLUMN deleted_at timestamptz,
    ADD COLUMN suspended_at timestamptz;

---- create above / drop below ----

ALTER TABLE organizations
    DROP COLUMN suspended_at,
    DROP COLUMN deleted_at;

ALTER TABLE organization_memberships
    DROP COLUMN removed_at,
    DROP CONSTRAINT organization_memberships_role_valid;

UPDATE organization_memberships
SET role = 'developer'
WHERE role = 'admin';

UPDATE organization_memberships
SET role = 'owner'
FROM organizations
WHERE organization_memberships.organization_id = organizations.id
  AND organization_memberships.user_id = organizations.owner_user_id;

ALTER TABLE organization_memberships
    ADD CONSTRAINT organization_memberships_role_valid CHECK (
        role IN ('owner', 'developer', 'finance', 'customer_support')
    );

ALTER TABLE organizations
    DROP CONSTRAINT organizations_owner_user_id_fkey,
    DROP COLUMN owner_user_id;
