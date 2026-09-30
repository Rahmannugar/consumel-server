ALTER TABLE balance_operations
    ADD COLUMN requested_expires_at timestamptz,
    ADD COLUMN resulting_next_expires_at timestamptz;

CREATE TABLE entitlement_grants (
    id uuid PRIMARY KEY,
    balance_id uuid NOT NULL REFERENCES balances (id),
    source_type text NOT NULL,
    granted_quantity bigint NOT NULL,
    remaining_quantity bigint NOT NULL,
    expires_at timestamptz,
    created_by_balance_operation_id uuid REFERENCES balance_operations (id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT entitlement_grants_source_valid CHECK (
        source_type IN ('manual', 'opening_balance')
    ),
    CONSTRAINT entitlement_grants_quantity_positive CHECK (granted_quantity > 0),
    CONSTRAINT entitlement_grants_remaining_valid CHECK (
        remaining_quantity >= 0 AND remaining_quantity <= granted_quantity
    )
);

CREATE UNIQUE INDEX entitlement_grants_balance_operation_idx
    ON entitlement_grants (created_by_balance_operation_id)
    WHERE created_by_balance_operation_id IS NOT NULL;

CREATE INDEX entitlement_grants_spendable_idx
    ON entitlement_grants (balance_id, expires_at ASC NULLS LAST, created_at, id)
    INCLUDE (remaining_quantity)
    WHERE remaining_quantity > 0;

CREATE TABLE balance_operation_grant_allocations (
    balance_operation_id uuid NOT NULL REFERENCES balance_operations (id),
    entitlement_grant_id uuid NOT NULL REFERENCES entitlement_grants (id),
    quantity bigint NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (balance_operation_id, entitlement_grant_id),
    CONSTRAINT balance_operation_grant_allocations_quantity_positive CHECK (quantity > 0)
);

CREATE TABLE consumption_grant_allocations (
    consumption_operation_id uuid NOT NULL REFERENCES consumption_operations (id),
    entitlement_grant_id uuid NOT NULL REFERENCES entitlement_grants (id),
    quantity bigint NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (consumption_operation_id, entitlement_grant_id),
    CONSTRAINT consumption_grant_allocations_quantity_positive CHECK (quantity > 0)
);

INSERT INTO entitlement_grants (
    id,
    balance_id,
    source_type,
    granted_quantity,
    remaining_quantity,
    created_at,
    updated_at
)
SELECT
    id,
    id,
    'opening_balance',
    quantity,
    quantity,
    created_at,
    updated_at
FROM balances
WHERE quantity > 0;

---- create above / drop below ----

DROP TABLE consumption_grant_allocations;
DROP TABLE balance_operation_grant_allocations;
DROP TABLE entitlement_grants;
ALTER TABLE balance_operations
    DROP COLUMN resulting_next_expires_at,
    DROP COLUMN requested_expires_at;
