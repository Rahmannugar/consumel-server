-- Authlier owns this runtime table and creates it through its versioned
-- PostgreSQL migration. This declaration exists only so sqlc can type-check
-- Consumel's one-query account projection against Authlier v0.5.0.
CREATE TABLE authlier_users (
    id text PRIMARY KEY,
    email text NOT NULL UNIQUE,
    email_verified boolean NOT NULL DEFAULT false,
    webauthn_handle bytea NOT NULL UNIQUE,
    created_at timestamptz NOT NULL
);
