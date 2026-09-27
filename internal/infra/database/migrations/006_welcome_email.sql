ALTER TABLE email_deliveries
    DROP CONSTRAINT email_deliveries_template_valid,
    ADD CONSTRAINT email_deliveries_template_valid CHECK (
        template IN ('email_verification', 'password_reset', 'welcome')
    );

---- create above / drop below ----

ALTER TABLE email_deliveries
    DROP CONSTRAINT email_deliveries_template_valid,
    ADD CONSTRAINT email_deliveries_template_valid CHECK (
        template IN ('email_verification', 'password_reset')
    );
