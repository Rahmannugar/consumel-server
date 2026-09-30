-- name: CreateMeter :one
WITH selected_environment AS MATERIALIZED (
    SELECT project_id
    FROM project_environments
    WHERE project_environments.id = sqlc.arg(project_environment_id)
), resolved_meter AS (
    INSERT INTO meters (id, project_id, meter_key, name, description, meter_type)
    SELECT
        sqlc.arg(meter_id),
        project_id,
        sqlc.arg(meter_key),
        sqlc.arg(name),
        sqlc.narg(description),
        sqlc.arg(meter_type)
    FROM selected_environment
    ON CONFLICT (project_id, meter_key) DO UPDATE
    SET meter_key = EXCLUDED.meter_key
    WHERE meters.name = EXCLUDED.name
      AND meters.description IS NOT DISTINCT FROM EXCLUDED.description
      AND meters.meter_type = EXCLUDED.meter_type
    RETURNING meters.id, meters.project_id, meters.meter_key, meters.name,
        meters.description, meters.meter_type
), configuration AS (
    INSERT INTO project_environment_meters (project_environment_id, meter_id)
    SELECT sqlc.arg(project_environment_id), resolved_meter.id
    FROM resolved_meter
    RETURNING *
)
SELECT
    resolved_meter.id,
    resolved_meter.project_id,
    configuration.project_environment_id,
    resolved_meter.meter_key,
    resolved_meter.name,
    resolved_meter.description,
    resolved_meter.meter_type,
    configuration.created_at,
    configuration.updated_at
FROM resolved_meter
JOIN configuration ON configuration.meter_id = resolved_meter.id;

-- name: MeterByPublicKey :one
SELECT
    meters.id,
    meters.project_id,
    project_environment_meters.project_environment_id,
    meters.meter_key,
    meters.name,
    meters.description,
    meters.meter_type,
    project_environment_meters.created_at,
    project_environment_meters.updated_at
FROM project_environment_meters
JOIN meters ON meters.id = project_environment_meters.meter_id
WHERE project_environment_meters.project_environment_id = $1
  AND project_environment_meters.archived_at IS NULL
  AND meters.meter_key = $2;

-- name: ListMeters :many
SELECT
    meters.id,
    meters.project_id,
    project_environment_meters.project_environment_id,
    meters.meter_key,
    meters.name,
    meters.description,
    meters.meter_type,
    project_environment_meters.created_at,
    project_environment_meters.updated_at
FROM project_environment_meters
JOIN meters ON meters.id = project_environment_meters.meter_id
WHERE project_environment_meters.project_environment_id = sqlc.arg(project_environment_id)
  AND project_environment_meters.archived_at IS NULL
  AND (
      sqlc.narg(cursor_created_at)::timestamptz IS NULL
      OR (project_environment_meters.created_at, meters.id) < (
          sqlc.narg(cursor_created_at)::timestamptz,
          sqlc.narg(cursor_id)::uuid
      )
  )
ORDER BY project_environment_meters.created_at DESC, meters.id DESC
LIMIT sqlc.arg(page_size);

-- name: SearchMeters :many
WITH search_scope AS (
    SELECT project_id
    FROM project_environments
    WHERE id = sqlc.arg(project_environment_id)
), matching_meter_ids AS (
    SELECT meters.id
    FROM meters
    WHERE meters.project_id = (SELECT project_id FROM search_scope)
      AND meters.search_vector @@ websearch_to_tsquery('simple', sqlc.arg(search_query))

    UNION

    SELECT meters.id
    FROM meters
    WHERE meters.project_id = (SELECT project_id FROM search_scope)
      AND char_length(sqlc.arg(search_query)) >= 3
      AND meters.search_text LIKE '%' || lower(sqlc.arg(search_query)) || '%'
)
SELECT
    meters.id,
    meters.project_id,
    project_environment_meters.project_environment_id,
    meters.meter_key,
    meters.name,
    meters.description,
    meters.meter_type,
    project_environment_meters.created_at,
    project_environment_meters.updated_at
FROM project_environment_meters
JOIN meters ON meters.id = project_environment_meters.meter_id
JOIN matching_meter_ids ON matching_meter_ids.id = meters.id
WHERE project_environment_meters.project_environment_id = sqlc.arg(project_environment_id)
  AND project_environment_meters.archived_at IS NULL
  AND (
      sqlc.narg(cursor_created_at)::timestamptz IS NULL
      OR (project_environment_meters.created_at, meters.id) < (
          sqlc.narg(cursor_created_at)::timestamptz,
          sqlc.narg(cursor_id)::uuid
      )
  )
ORDER BY project_environment_meters.created_at DESC, meters.id DESC
LIMIT sqlc.arg(page_size);
