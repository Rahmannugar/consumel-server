ALTER TABLE users
    RENAME COLUMN clerk_user_id TO authlier_subject_id;

ALTER TABLE users
    RENAME CONSTRAINT users_clerk_user_id_key TO users_authlier_subject_id_key;

ALTER TABLE users
    RENAME CONSTRAINT users_clerk_user_id_not_blank TO users_authlier_subject_id_not_blank;

CREATE INDEX organization_memberships_active_user_idx
    ON organization_memberships (user_id, organization_id)
    WHERE status = 'active' AND removed_at IS NULL;

---- create above / drop below ----

DROP INDEX organization_memberships_active_user_idx;

ALTER TABLE users
    RENAME CONSTRAINT users_authlier_subject_id_not_blank TO users_clerk_user_id_not_blank;

ALTER TABLE users
    RENAME CONSTRAINT users_authlier_subject_id_key TO users_clerk_user_id_key;

ALTER TABLE users
    RENAME COLUMN authlier_subject_id TO clerk_user_id;
