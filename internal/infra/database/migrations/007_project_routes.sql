ALTER TABLE projects
    ADD COLUMN slug text;

WITH normalized_projects AS (
    SELECT
        id,
        organization_id,
        COALESCE(
            NULLIF(
                trim(BOTH '-' FROM lower(regexp_replace(name, '[^a-zA-Z0-9]+', '-', 'g'))),
                ''
            ),
            'project'
        ) AS base_slug
    FROM projects
), ranked_projects AS (
    SELECT
        id,
        base_slug,
        row_number() OVER (
            PARTITION BY organization_id, left(base_slug, 120)
            ORDER BY id
        ) AS slug_position
    FROM normalized_projects
)
UPDATE projects
SET slug = CASE
    WHEN ranked_projects.slug_position = 1 THEN left(ranked_projects.base_slug, 120)
    ELSE left(ranked_projects.base_slug, 111) || '-' || left(replace(projects.id::text, '-', ''), 8)
END
FROM ranked_projects
WHERE ranked_projects.id = projects.id;

ALTER TABLE projects
    ALTER COLUMN slug SET NOT NULL,
    ADD CONSTRAINT projects_slug_not_blank CHECK (length(btrim(slug)) > 0),
    ADD CONSTRAINT projects_slug_length CHECK (length(slug) <= 120),
    ADD CONSTRAINT projects_name_length CHECK (length(name) <= 120);

CREATE UNIQUE INDEX projects_organization_name_unique_idx
    ON projects (organization_id, lower(btrim(name)));

CREATE UNIQUE INDEX projects_organization_slug_unique_idx
    ON projects (organization_id, slug);

CREATE UNIQUE INDEX organization_memberships_current_user_unique_idx
    ON organization_memberships (user_id)
    WHERE removed_at IS NULL;

---- create above / drop below ----

DROP INDEX organization_memberships_current_user_unique_idx;
DROP INDEX projects_organization_slug_unique_idx;
DROP INDEX projects_organization_name_unique_idx;

ALTER TABLE projects
    DROP CONSTRAINT projects_name_length,
    DROP CONSTRAINT projects_slug_length,
    DROP CONSTRAINT projects_slug_not_blank,
    DROP COLUMN slug;
