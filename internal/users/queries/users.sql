-- name: ResolveUserByAuthlierSubjectID :one
WITH inserted AS (
    INSERT INTO users (id, authlier_subject_id)
    VALUES ($1, $2)
    ON CONFLICT (authlier_subject_id) DO NOTHING
    RETURNING id, authlier_subject_id, created_at
)
SELECT id, authlier_subject_id, created_at
FROM inserted
UNION ALL
SELECT id, authlier_subject_id, created_at
FROM users
WHERE authlier_subject_id = $2
LIMIT 1;

-- name: GetUserByAuthlierSubjectID :one
SELECT id, authlier_subject_id, created_at
FROM users
WHERE authlier_subject_id = $1;
