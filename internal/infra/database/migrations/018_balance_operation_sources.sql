ALTER TABLE balance_operations
    ADD COLUMN source_type text,
    ADD COLUMN source_user_id uuid REFERENCES users (id),
    ADD COLUMN source_api_key_id uuid REFERENCES project_api_keys (id);

UPDATE balance_operations SET source_type = 'legacy';

ALTER TABLE balance_operations
    ALTER COLUMN source_type SET NOT NULL,
    ADD CONSTRAINT balance_operations_source_valid CHECK (
        (source_type = 'legacy' AND source_user_id IS NULL AND source_api_key_id IS NULL)
        OR (source_type = 'dashboard_user' AND source_user_id IS NOT NULL AND source_api_key_id IS NULL)
        OR (source_type = 'api_key' AND source_user_id IS NULL AND source_api_key_id IS NOT NULL)
    );

ALTER TABLE consumption_operations
    ADD COLUMN source_api_key_id uuid REFERENCES project_api_keys (id);

CREATE INDEX balance_operations_balance_created_idx
    ON balance_operations (balance_id, created_at DESC, id DESC)
    WHERE balance_id IS NOT NULL;

CREATE INDEX entitlement_grants_balance_created_idx
    ON entitlement_grants (balance_id, created_at DESC, id DESC);

CREATE INDEX consumption_operations_balance_activity_idx
    ON consumption_operations (balance_id, created_at DESC, id DESC)
    WHERE balance_id IS NOT NULL
      AND status = 'accepted'
      AND balance_debited > 0;

---- create above / drop below ----

DROP INDEX consumption_operations_balance_activity_idx;
DROP INDEX entitlement_grants_balance_created_idx;
DROP INDEX balance_operations_balance_created_idx;
ALTER TABLE balance_operations
    DROP CONSTRAINT balance_operations_source_valid,
    DROP COLUMN source_api_key_id,
    DROP COLUMN source_user_id,
    DROP COLUMN source_type;
ALTER TABLE consumption_operations
    DROP COLUMN source_api_key_id;
