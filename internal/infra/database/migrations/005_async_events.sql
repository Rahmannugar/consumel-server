CREATE TABLE outbox_events (
    id uuid PRIMARY KEY,
    event_type text NOT NULL,
    aggregate_type text NOT NULL,
    aggregate_id uuid NOT NULL,
    payload jsonb NOT NULL,
    trace_context jsonb NOT NULL DEFAULT '{}'::jsonb,
    occurred_at timestamptz NOT NULL DEFAULT now(),
    available_at timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz,
    publish_claim_id uuid,
    publish_claimed_at timestamptz,
    publish_attempts integer NOT NULL DEFAULT 0,
    last_publish_error text,
    CONSTRAINT outbox_events_event_type_not_blank CHECK (length(btrim(event_type)) > 0),
    CONSTRAINT outbox_events_aggregate_type_not_blank CHECK (length(btrim(aggregate_type)) > 0),
    CONSTRAINT outbox_events_payload_object CHECK (jsonb_typeof(payload) = 'object'),
    CONSTRAINT outbox_events_trace_context_object CHECK (jsonb_typeof(trace_context) = 'object'),
    CONSTRAINT outbox_events_publish_attempts_nonnegative CHECK (publish_attempts >= 0)
);

CREATE INDEX outbox_events_unpublished_idx
    ON outbox_events (available_at, id)
    WHERE published_at IS NULL;

CREATE TABLE email_deliveries (
    id uuid PRIMARY KEY,
    template text NOT NULL,
    encrypted_payload bytea NOT NULL,
    payload_nonce bytea NOT NULL,
    payload_key_version smallint NOT NULL DEFAULT 1,
    expires_at timestamptz NOT NULL,
    status text NOT NULL DEFAULT 'pending',
    attempts integer NOT NULL DEFAULT 0,
    processing_started_at timestamptz,
    processing_claim_id uuid,
    next_attempt_at timestamptz NOT NULL DEFAULT now(),
    last_error text,
    delivered_at timestamptz,
    provider_message_id text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT email_deliveries_template_valid CHECK (
        template IN ('email_verification', 'password_reset')
    ),
    CONSTRAINT email_deliveries_payload_not_empty CHECK (octet_length(encrypted_payload) > 0),
    CONSTRAINT email_deliveries_nonce_length CHECK (octet_length(payload_nonce) = 12),
    CONSTRAINT email_deliveries_key_version_positive CHECK (payload_key_version > 0),
    CONSTRAINT email_deliveries_status_valid CHECK (
        status IN ('pending', 'processing', 'retrying', 'delivered', 'failed', 'expired')
    ),
    CONSTRAINT email_deliveries_attempts_nonnegative CHECK (attempts >= 0)
);

CREATE OR REPLACE FUNCTION notify_outbox_event() RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify('outbox_events', NEW.id::text);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER outbox_events_notify_after_insert
AFTER INSERT ON outbox_events
FOR EACH ROW EXECUTE FUNCTION notify_outbox_event();

---- create above / drop below ----

DROP TRIGGER outbox_events_notify_after_insert ON outbox_events;
DROP FUNCTION notify_outbox_event();
DROP TABLE email_deliveries;
DROP TABLE outbox_events;
