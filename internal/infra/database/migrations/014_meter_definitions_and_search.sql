CREATE EXTENSION IF NOT EXISTS btree_gin;

DO $$
BEGIN
    IF EXISTS (
        SELECT meter_id
        FROM project_environment_meters
        GROUP BY meter_id
        HAVING count(DISTINCT ROW(name, description, meter_type)) > 1
    ) THEN
        RAISE EXCEPTION 'meter definitions differ across project environments';
    END IF;
END $$;

ALTER TABLE meters
    ADD COLUMN name text,
    ADD COLUMN description text,
    ADD COLUMN meter_type text,
    ADD COLUMN updated_at timestamptz;

UPDATE meters
SET
    name = configuration.name,
    description = configuration.description,
    meter_type = configuration.meter_type,
    updated_at = meters.created_at
FROM (
    SELECT DISTINCT ON (meter_id)
        meter_id,
        name,
        description,
        meter_type
    FROM project_environment_meters
    ORDER BY meter_id, created_at, project_environment_id
) AS configuration
WHERE configuration.meter_id = meters.id;

ALTER TABLE meters
    ALTER COLUMN name SET NOT NULL,
    ALTER COLUMN meter_type SET NOT NULL,
    ALTER COLUMN updated_at SET DEFAULT now(),
    ALTER COLUMN updated_at SET NOT NULL,
    ADD CONSTRAINT meters_name_not_blank CHECK (length(btrim(name)) > 0),
    ADD CONSTRAINT meters_name_length CHECK (length(name) <= 120),
    ADD CONSTRAINT meters_description_valid CHECK (
        description IS NULL OR (length(btrim(description)) > 0 AND length(description) <= 500)
    ),
    ADD CONSTRAINT meters_type_valid CHECK (meter_type IN ('prepaid', 'postpaid', 'hybrid'));

DROP INDEX customers_search_idx;
DROP INDEX meters_key_search_idx;
DROP INDEX project_environment_meters_name_search_idx;

ALTER TABLE project_environment_meters
    DROP COLUMN name,
    DROP COLUMN description,
    DROP COLUMN meter_type;

ALTER TABLE customers
    ADD COLUMN search_text text GENERATED ALWAYS AS (
        lower(customer_id || ' ' || COALESCE(name, '') || ' ' || COALESCE(email, ''))
    ) STORED,
    ADD COLUMN search_vector tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('simple', customer_id), 'A') ||
        setweight(to_tsvector('simple', COALESCE(name, '')), 'B') ||
        setweight(to_tsvector('simple', COALESCE(email, '')), 'A')
    ) STORED;

ALTER TABLE meters
    ADD COLUMN search_text text GENERATED ALWAYS AS (
        lower(meter_key || ' ' || name)
    ) STORED,
    ADD COLUMN search_vector tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('simple', meter_key), 'A') ||
        setweight(to_tsvector('simple', name), 'B')
    ) STORED;

CREATE INDEX customers_environment_search_vector_idx
    ON customers USING gin (project_environment_id, search_vector);

CREATE INDEX customers_environment_search_text_idx
    ON customers USING gin (project_environment_id, search_text gin_trgm_ops);

CREATE INDEX meters_project_search_vector_idx
    ON meters USING gin (project_id, search_vector);

CREATE INDEX meters_project_search_text_idx
    ON meters USING gin (project_id, search_text gin_trgm_ops);

---- create above / drop below ----

DROP INDEX meters_project_search_text_idx;
DROP INDEX meters_project_search_vector_idx;
DROP INDEX customers_environment_search_text_idx;
DROP INDEX customers_environment_search_vector_idx;

ALTER TABLE meters
    DROP COLUMN search_vector,
    DROP COLUMN search_text;

ALTER TABLE customers
    DROP COLUMN search_vector,
    DROP COLUMN search_text;

ALTER TABLE project_environment_meters
    ADD COLUMN name text,
    ADD COLUMN description text,
    ADD COLUMN meter_type text;

UPDATE project_environment_meters
SET
    name = meters.name,
    description = meters.description,
    meter_type = meters.meter_type
FROM meters
WHERE meters.id = project_environment_meters.meter_id;

ALTER TABLE project_environment_meters
    ALTER COLUMN name SET NOT NULL,
    ALTER COLUMN meter_type SET NOT NULL,
    ADD CONSTRAINT project_environment_meters_name_not_blank CHECK (length(btrim(name)) > 0),
    ADD CONSTRAINT project_environment_meters_name_length CHECK (length(name) <= 120),
    ADD CONSTRAINT project_environment_meters_description_valid CHECK (
        description IS NULL OR (length(btrim(description)) > 0 AND length(description) <= 500)
    ),
    ADD CONSTRAINT project_environment_meters_type_valid CHECK (
        meter_type IN ('prepaid', 'postpaid', 'hybrid')
    );

ALTER TABLE meters
    DROP COLUMN updated_at,
    DROP COLUMN meter_type,
    DROP COLUMN description,
    DROP COLUMN name;

CREATE INDEX customers_search_idx
    ON customers USING gin (
        (customer_id || ' ' || COALESCE(name, '') || ' ' || COALESCE(email, '')) gin_trgm_ops
    );

CREATE INDEX meters_key_search_idx
    ON meters USING gin (meter_key gin_trgm_ops);

CREATE INDEX project_environment_meters_name_search_idx
    ON project_environment_meters USING gin (name gin_trgm_ops)
    WHERE archived_at IS NULL;
