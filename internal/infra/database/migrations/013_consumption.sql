CREATE TABLE consumption_operations (
    id uuid PRIMARY KEY,
    project_environment_id uuid NOT NULL REFERENCES project_environments (id),
    idempotency_key uuid NOT NULL,
    request_customer_id text NOT NULL,
    request_meter_key text NOT NULL,
    requested_quantity bigint NOT NULL,
    customer_id uuid NOT NULL,
    meter_id uuid NOT NULL,
    meter_type text NOT NULL,
    status text NOT NULL DEFAULT 'pending',
    denial_reason text,
    balance_id uuid REFERENCES balances (id),
    balance_debited bigint NOT NULL DEFAULT 0,
    resulting_balance bigint,
    billable boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT consumption_operations_quantity_positive CHECK (requested_quantity > 0),
    CONSTRAINT consumption_operations_meter_type_valid CHECK (
        meter_type IN ('prepaid', 'postpaid', 'hybrid')
    ),
    CONSTRAINT consumption_operations_status_valid CHECK (
        status IN ('pending', 'accepted', 'denied')
    ),
    CONSTRAINT consumption_operations_denial_valid CHECK (
        (status = 'denied' AND denial_reason = 'insufficient_balance')
        OR (status <> 'denied' AND denial_reason IS NULL)
    ),
    CONSTRAINT consumption_operations_balance_debited_valid CHECK (
        balance_debited >= 0 AND balance_debited <= requested_quantity
    ),
    CONSTRAINT consumption_operations_result_valid CHECK (
        (status = 'pending' AND balance_id IS NULL AND balance_debited = 0 AND resulting_balance IS NULL AND billable = false)
        OR (status = 'denied' AND balance_debited = 0 AND billable = false)
        OR status = 'accepted'
    ),
    CONSTRAINT consumption_operations_customer_fk FOREIGN KEY (project_environment_id, customer_id)
        REFERENCES customers (project_environment_id, id),
    CONSTRAINT consumption_operations_meter_fk FOREIGN KEY (project_environment_id, meter_id)
        REFERENCES project_environment_meters (project_environment_id, meter_id),
    CONSTRAINT consumption_operations_environment_idempotency_unique
        UNIQUE (project_environment_id, idempotency_key)
);

CREATE INDEX consumption_operations_environment_created_idx
    ON consumption_operations (project_environment_id, created_at DESC, id DESC);

---- create above / drop below ----

DROP TABLE consumption_operations;
