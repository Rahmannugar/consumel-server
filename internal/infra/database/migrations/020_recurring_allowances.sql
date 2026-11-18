-- Lock waiters use one entitlement clock captured after acquiring the balance
-- lock. A transaction begun before renewal cannot spend an already-carried lot.
CREATE FUNCTION consumption_entitlement_time() RETURNS timestamptz
LANGUAGE sql STABLE AS $$
 SELECT COALESCE(NULLIF(current_setting('consumel.entitlement_at', true), '')::timestamptz, statement_timestamp());
$$;

CREATE TABLE recurring_allowances (
    id uuid PRIMARY KEY,
    balance_id uuid NOT NULL UNIQUE REFERENCES balances(id),
    anchor_at timestamptz NOT NULL,
    next_period_at timestamptz NOT NULL,
    version bigint NOT NULL CHECK (version > 0),
    retry_at timestamptz NOT NULL DEFAULT now(),
    retry_attempts integer NOT NULL DEFAULT 0 CHECK (retry_attempts >= 0),
    CONSTRAINT recurring_allowances_anchor_midnight CHECK (anchor_at = date_trunc('day', anchor_at AT TIME ZONE 'UTC') AT TIME ZONE 'UTC')
);
CREATE INDEX recurring_allowances_due_idx ON recurring_allowances(next_period_at, retry_at, id);
CREATE TABLE allowance_revisions (
    allowance_id uuid NOT NULL REFERENCES recurring_allowances(id),
    version bigint NOT NULL CHECK (version > 0),
    effective_at timestamptz NOT NULL,
    quantity bigint NOT NULL CHECK (quantity > 0),
    rollover_cap bigint NOT NULL CHECK (rollover_cap >= 0 AND rollover_cap <= 9223372036854775807 - quantity),
    enabled boolean NOT NULL,
    source_type text NOT NULL,
    source_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (allowance_id, version),
    CONSTRAINT allowance_revisions_source_valid CHECK (source_type IN ('dashboard_user','api_key'))
);
CREATE INDEX allowance_revisions_effective_idx ON allowance_revisions(allowance_id, effective_at DESC, version DESC);
ALTER TABLE entitlement_grants DROP CONSTRAINT entitlement_grants_source_valid;
ALTER TABLE entitlement_grants ADD CONSTRAINT entitlement_grants_source_valid CHECK (source_type IN ('manual','opening_balance','recurring_allowance'));
CREATE TABLE allowance_periods (
    allowance_id uuid NOT NULL REFERENCES recurring_allowances(id),
    period_start timestamptz NOT NULL,
    period_end timestamptz NOT NULL,
    revision bigint NOT NULL,
    quantity bigint NOT NULL CHECK (quantity >= 0),
    carried_quantity bigint NOT NULL CHECK (carried_quantity >= 0),
    skipped_periods integer NOT NULL CHECK (skipped_periods >= 0),
    grant_id uuid UNIQUE REFERENCES entitlement_grants(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (allowance_id, period_start),
    FOREIGN KEY (allowance_id, revision) REFERENCES allowance_revisions(allowance_id, version),
    CHECK (period_end > period_start),
    CHECK (carried_quantity <= quantity)
);
---- create above / drop below ----
DROP TABLE allowance_periods;
DROP TABLE allowance_revisions;
DROP TABLE recurring_allowances;
DROP FUNCTION consumption_entitlement_time();
ALTER TABLE entitlement_grants DROP CONSTRAINT entitlement_grants_source_valid;
ALTER TABLE entitlement_grants ADD CONSTRAINT entitlement_grants_source_valid CHECK (source_type IN ('manual','opening_balance'));
