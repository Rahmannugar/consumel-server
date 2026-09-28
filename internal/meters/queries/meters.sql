-- name: CreateMeter :one
WITH selected_environment AS MATERIALIZED (
    SELECT project_id
    FROM project_environments
    WHERE project_environments.id = sqlc.arg(project_environment_id)
), resolved_meter AS (
    INSERT INTO meters (id, project_id, meter_key)
    SELECT sqlc.arg(meter_id), project_id, sqlc.arg(meter_key)
    FROM selected_environment
    ON CONFLICT (project_id, meter_key) DO UPDATE
    SET meter_key = EXCLUDED.meter_key
    RETURNING meters.id, meters.project_id, meters.meter_key
), configuration AS (
    INSERT INTO project_environment_meters (
        project_environment_id,
        meter_id,
        name,
        description,
        meter_type
    )
    SELECT
        sqlc.arg(project_environment_id),
        resolved_meter.id,
        sqlc.arg(name),
        sqlc.narg(description),
        sqlc.arg(meter_type)
    FROM resolved_meter
    RETURNING *
)
SELECT
    resolved_meter.id,
    resolved_meter.project_id,
    configuration.project_environment_id,
    resolved_meter.meter_key,
    configuration.name,
    configuration.description,
    configuration.meter_type,
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
    project_environment_meters.name,
    project_environment_meters.description,
    project_environment_meters.meter_type,
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
    project_environment_meters.name,
    project_environment_meters.description,
    project_environment_meters.meter_type,
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
