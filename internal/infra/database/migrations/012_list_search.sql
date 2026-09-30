CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX customers_search_idx
    ON customers USING gin (
        (customer_id || ' ' || COALESCE(name, '') || ' ' || COALESCE(email, '')) gin_trgm_ops
    );

CREATE INDEX meters_key_search_idx
    ON meters USING gin (meter_key gin_trgm_ops);

CREATE INDEX project_environment_meters_name_search_idx
    ON project_environment_meters USING gin (name gin_trgm_ops)
    WHERE archived_at IS NULL;

---- create above / drop below ----

DROP INDEX project_environment_meters_name_search_idx;
DROP INDEX meters_key_search_idx;
DROP INDEX customers_search_idx;
DROP EXTENSION pg_trgm;
