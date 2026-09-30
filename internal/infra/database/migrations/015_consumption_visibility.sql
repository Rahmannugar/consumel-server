ALTER TABLE consumption_operations
    ADD COLUMN replay_count bigint NOT NULL DEFAULT 0,
    ADD COLUMN last_replayed_at timestamptz,
    ADD CONSTRAINT consumption_operations_replay_count_valid CHECK (replay_count >= 0),
    ADD CONSTRAINT consumption_operations_replay_time_valid CHECK (
        (replay_count = 0 AND last_replayed_at IS NULL)
        OR (replay_count > 0 AND last_replayed_at IS NOT NULL)
    );

---- create above / drop below ----

ALTER TABLE consumption_operations
    DROP COLUMN last_replayed_at,
    DROP COLUMN replay_count;
