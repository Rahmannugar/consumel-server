CREATE TABLE meters (
    id uuid PRIMARY KEY,
    project_id uuid NOT NULL REFERENCES projects (id),
    meter_key text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT meters_key_not_blank CHECK (length(btrim(meter_key)) > 0),
    CONSTRAINT meters_key_length CHECK (length(meter_key) <= 120),
    CONSTRAINT meters_key_format CHECK (meter_key ~ '^[a-z][a-z0-9_-]*$'),
    UNIQUE (project_id, meter_key)
);

CREATE TABLE project_environment_meters (
    project_environment_id uuid NOT NULL REFERENCES project_environments (id),
    meter_id uuid NOT NULL REFERENCES meters (id),
    name text NOT NULL,
    description text,
    meter_type text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    archived_at timestamptz,
    PRIMARY KEY (project_environment_id, meter_id),
    CONSTRAINT project_environment_meters_name_not_blank CHECK (length(btrim(name)) > 0),
    CONSTRAINT project_environment_meters_name_length CHECK (length(name) <= 120),
    CONSTRAINT project_environment_meters_description_valid CHECK (
        description IS NULL OR (length(btrim(description)) > 0 AND length(description) <= 500)
    ),
    CONSTRAINT project_environment_meters_type_valid CHECK (
        meter_type IN ('prepaid', 'postpaid', 'hybrid')
    )
);

CREATE INDEX project_environment_meters_list_idx
    ON project_environment_meters (project_environment_id, created_at DESC, meter_id DESC)
    WHERE archived_at IS NULL;

---- create above / drop below ----

DROP TABLE project_environment_meters;
DROP TABLE meters;
