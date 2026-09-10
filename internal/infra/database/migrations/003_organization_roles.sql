CREATE TABLE organization_roles (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations (id),
    name text NOT NULL,
    system_key text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    UNIQUE (organization_id, id),
    CONSTRAINT organization_roles_name_not_blank CHECK (length(btrim(name)) > 0),
    CONSTRAINT organization_roles_system_key_valid CHECK (
        system_key IS NULL OR system_key IN ('admin', 'developer')
    )
);

CREATE UNIQUE INDEX organization_roles_system_key_unique
    ON organization_roles (organization_id, system_key)
    WHERE system_key IS NOT NULL;

CREATE UNIQUE INDEX organization_roles_active_name_unique
    ON organization_roles (organization_id, lower(name))
    WHERE deleted_at IS NULL;

INSERT INTO organization_roles (id, organization_id, name, system_key)
SELECT md5(id::text || ':admin')::uuid, id, 'Admin', 'admin'
FROM organizations;

INSERT INTO organization_roles (id, organization_id, name, system_key)
SELECT md5(id::text || ':developer')::uuid, id, 'Developer', 'developer'
FROM organizations;

ALTER TABLE organization_memberships
    ADD COLUMN role_id uuid;

UPDATE organization_memberships
SET role_id = organization_roles.id
FROM organization_roles
WHERE organization_roles.organization_id = organization_memberships.organization_id
  AND organization_roles.system_key = CASE
      WHEN organization_memberships.role = 'admin' THEN 'admin'
      ELSE 'developer'
  END;

ALTER TABLE organization_memberships
    ALTER COLUMN role_id SET NOT NULL,
    ADD CONSTRAINT organization_memberships_role_fkey
        FOREIGN KEY (organization_id, role_id)
        REFERENCES organization_roles (organization_id, id),
    DROP COLUMN role;

---- create above / drop below ----

ALTER TABLE organization_memberships
    ADD COLUMN role text;

UPDATE organization_memberships
SET role = COALESCE(organization_roles.system_key, 'developer')
FROM organization_roles
WHERE organization_roles.id = organization_memberships.role_id;

ALTER TABLE organization_memberships
    ALTER COLUMN role SET NOT NULL,
    ADD CONSTRAINT organization_memberships_role_valid CHECK (
        role IN ('admin', 'developer')
    ),
    DROP CONSTRAINT organization_memberships_role_fkey,
    DROP COLUMN role_id;

DROP TABLE organization_roles;
