DROP INDEX consumption_operations_environment_created_idx;

CREATE INDEX consumption_operations_environment_created_idx
    ON consumption_operations (project_environment_id, created_at DESC, id DESC)
    WHERE status <> 'pending';

CREATE INDEX consumption_operations_environment_status_created_idx
    ON consumption_operations (project_environment_id, status, created_at DESC, id DESC)
    WHERE status <> 'pending';

---- create above / drop below ----

DROP INDEX consumption_operations_environment_status_created_idx;
DROP INDEX consumption_operations_environment_created_idx;

CREATE INDEX consumption_operations_environment_created_idx
    ON consumption_operations (project_environment_id, created_at DESC, id DESC);
