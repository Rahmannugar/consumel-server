CREATE TABLE customers (
    id uuid PRIMARY KEY,
    project_environment_id uuid NOT NULL REFERENCES project_environments (id),
    customer_id text NOT NULL,
    name text,
    email text,
    metadata_plan text,
    metadata_country text,
    metadata_location text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT customers_customer_id_not_blank CHECK (length(btrim(customer_id)) > 0),
    CONSTRAINT customers_customer_id_length CHECK (length(customer_id) <= 255),
    CONSTRAINT customers_name_valid CHECK (
        name IS NULL OR (length(btrim(name)) > 0 AND length(name) <= 200)
    ),
    CONSTRAINT customers_email_valid CHECK (
        email IS NULL OR (length(btrim(email)) > 0 AND length(email) <= 320)
    ),
    CONSTRAINT customers_metadata_plan_valid CHECK (
        metadata_plan IS NULL OR (length(btrim(metadata_plan)) > 0 AND length(metadata_plan) <= 120)
    ),
    CONSTRAINT customers_metadata_country_valid CHECK (
        metadata_country IS NULL OR (length(btrim(metadata_country)) > 0 AND length(metadata_country) <= 120)
    ),
    CONSTRAINT customers_metadata_location_valid CHECK (
        metadata_location IS NULL OR (length(btrim(metadata_location)) > 0 AND length(metadata_location) <= 120)
    ),
    UNIQUE (project_environment_id, customer_id)
);

CREATE INDEX customers_environment_created_idx
    ON customers (project_environment_id, created_at DESC, id DESC);

---- create above / drop below ----

DROP TABLE customers;
