CREATE TABLE project_api_keys (
    id uuid PRIMARY KEY,
    project_environment_id uuid NOT NULL REFERENCES project_environments (id),
    key_hash bytea NOT NULL UNIQUE,
    key_prefix text NOT NULL,
    last_four text NOT NULL,
    created_by_user_id uuid NOT NULL REFERENCES users (id),
    created_at timestamptz NOT NULL DEFAULT now(),
    last_used_at timestamptz,
    revoked_at timestamptz,
    revoked_by_user_id uuid REFERENCES users (id),
    CONSTRAINT project_api_keys_hash_length CHECK (octet_length(key_hash) = 32),
    CONSTRAINT project_api_keys_prefix_valid CHECK (key_prefix IN ('cm_test_', 'cm_live_')),
    CONSTRAINT project_api_keys_last_four_length CHECK (length(last_four) = 4),
    CONSTRAINT project_api_keys_revocation_consistent CHECK (
        (revoked_at IS NULL AND revoked_by_user_id IS NULL)
        OR (revoked_at IS NOT NULL AND revoked_by_user_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX project_api_keys_active_environment_unique_idx
    ON project_api_keys (project_environment_id)
    WHERE revoked_at IS NULL;

---- create above / drop below ----

DROP TABLE project_api_keys;
