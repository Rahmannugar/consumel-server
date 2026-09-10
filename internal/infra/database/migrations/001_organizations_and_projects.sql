CREATE TABLE users (
    id uuid PRIMARY KEY,
    clerk_user_id text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_clerk_user_id_not_blank CHECK (length(btrim(clerk_user_id)) > 0)
);

CREATE TABLE organizations (
    id uuid PRIMARY KEY,
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT organizations_name_not_blank CHECK (length(btrim(name)) > 0)
);

CREATE TABLE organization_memberships (
    organization_id uuid NOT NULL REFERENCES organizations (id),
    user_id uuid NOT NULL REFERENCES users (id),
    role text NOT NULL,
    status text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (organization_id, user_id),
    CONSTRAINT organization_memberships_role_valid CHECK (
        role IN ('owner', 'developer', 'finance', 'customer_support')
    ),
    CONSTRAINT organization_memberships_status_valid CHECK (status IN ('active', 'suspended'))
);

CREATE INDEX organization_memberships_user_id_idx ON organization_memberships (user_id);

CREATE TABLE projects (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations (id),
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT projects_name_not_blank CHECK (length(btrim(name)) > 0)
);

CREATE INDEX projects_organization_id_idx ON projects (organization_id);

CREATE TABLE project_environments (
    id uuid PRIMARY KEY,
    project_id uuid NOT NULL REFERENCES projects (id),
    environment text NOT NULL,
    activated_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, environment),
    CONSTRAINT project_environments_environment_valid CHECK (
        environment IN ('sandbox', 'live')
    ),
    CONSTRAINT project_environments_sandbox_active CHECK (
        environment <> 'sandbox' OR activated_at IS NOT NULL
    )
);

---- create above / drop below ----

DROP TABLE project_environments;
DROP TABLE projects;
DROP TABLE organization_memberships;
DROP TABLE organizations;
DROP TABLE users;
