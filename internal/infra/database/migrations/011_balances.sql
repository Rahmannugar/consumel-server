ALTER TABLE customers
    ADD CONSTRAINT customers_environment_internal_id_unique
    UNIQUE (project_environment_id, id);

CREATE TABLE balances (
    id uuid PRIMARY KEY,
    project_environment_id uuid NOT NULL,
    customer_id uuid NOT NULL,
    meter_id uuid NOT NULL,
    quantity bigint NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT balances_quantity_nonnegative CHECK (quantity >= 0),
    CONSTRAINT balances_customer_fk FOREIGN KEY (project_environment_id, customer_id)
        REFERENCES customers (project_environment_id, id),
    CONSTRAINT balances_meter_fk FOREIGN KEY (project_environment_id, meter_id)
        REFERENCES project_environment_meters (project_environment_id, meter_id),
    CONSTRAINT balances_environment_customer_meter_unique
        UNIQUE (project_environment_id, customer_id, meter_id)
);

CREATE TABLE balance_operations (
    id uuid PRIMARY KEY,
    project_environment_id uuid NOT NULL REFERENCES project_environments (id),
    idempotency_key uuid,
    operation_type text NOT NULL,
    request_customer_id text NOT NULL,
    request_meter_key text NOT NULL,
    requested_quantity bigint NOT NULL,
    customer_id uuid NOT NULL,
    meter_id uuid NOT NULL,
    balance_id uuid REFERENCES balances (id),
    resulting_quantity bigint,
    resulting_created_at timestamptz,
    resulting_updated_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT balance_operations_type_valid CHECK (operation_type IN ('add', 'set')),
    CONSTRAINT balance_operations_requested_quantity_valid CHECK (
        (operation_type = 'add' AND requested_quantity > 0)
        OR (operation_type = 'set' AND requested_quantity >= 0)
    ),
    CONSTRAINT balance_operations_idempotency_valid CHECK (
        (operation_type = 'add' AND idempotency_key IS NOT NULL)
        OR (operation_type = 'set' AND idempotency_key IS NULL)
    ),
    CONSTRAINT balance_operations_result_valid CHECK (
        (balance_id IS NULL AND resulting_quantity IS NULL AND resulting_created_at IS NULL AND resulting_updated_at IS NULL)
        OR (balance_id IS NOT NULL AND resulting_quantity IS NOT NULL AND resulting_created_at IS NOT NULL AND resulting_updated_at IS NOT NULL)
    ),
    CONSTRAINT balance_operations_customer_fk FOREIGN KEY (project_environment_id, customer_id)
        REFERENCES customers (project_environment_id, id),
    CONSTRAINT balance_operations_meter_fk FOREIGN KEY (project_environment_id, meter_id)
        REFERENCES project_environment_meters (project_environment_id, meter_id)
);

CREATE UNIQUE INDEX balance_operations_environment_idempotency_idx
    ON balance_operations (project_environment_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

---- create above / drop below ----

DROP TABLE balance_operations;
DROP TABLE balances;
ALTER TABLE customers DROP CONSTRAINT customers_environment_internal_id_unique;
