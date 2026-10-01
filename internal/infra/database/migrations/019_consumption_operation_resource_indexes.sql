CREATE INDEX consumption_operations_environment_customer_created_idx
    ON consumption_operations (project_environment_id, request_customer_id, created_at DESC, id DESC)
    WHERE status <> 'pending';

CREATE INDEX consumption_operations_environment_meter_created_idx
    ON consumption_operations (project_environment_id, request_meter_key, created_at DESC, id DESC)
    WHERE status <> 'pending';

---- create above / drop below ----

DROP INDEX consumption_operations_environment_meter_created_idx;
DROP INDEX consumption_operations_environment_customer_created_idx;
