-- =========================
-- JOBS
-- =========================

-- name: CreateJob :one
INSERT INTO jobs (
    id,
    name
) VALUES (
    $1, $2
)
RETURNING id;

-- name: GetJob :one
SELECT
    id,
    name,
    created_at,
    updated_at
FROM jobs
WHERE id = $1;

-- name: UpdateJobName :one
UPDATE jobs
SET
    name = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING id;

-- name: DeleteJob :exec
DELETE FROM jobs
WHERE id = $1;


-- =========================
-- SCHEDULE
-- =========================

-- name: CreateJobSchedule :exec
INSERT INTO job_schedules (
    job_id,
    repeat_interval_sec,
    target_runs,
    last_run_at,
    next_run_at
) VALUES (
    $1, $2, $3, NULL, $4
);

-- name: GetJobSchedule :one
SELECT
    job_id,
    status,
    repeat_interval_sec,
    scheduled_runs,
    target_runs,
    last_run_at,
    next_run_at,
    created_at,
    updated_at
FROM job_schedules
WHERE job_id = $1;

-- name: PatchJobSchedule :one
UPDATE job_schedules
SET
    repeat_interval_sec = COALESCE(sqlc.narg(repeat_interval_sec), repeat_interval_sec),
    target_runs = COALESCE(sqlc.narg(target_runs), target_runs),
    next_run_at = CASE
        WHEN sqlc.arg(set_next_run_at)::bool THEN sqlc.arg(next_run_at)
        ELSE next_run_at
    END,
    status = CASE 
        WHEN sqlc.narg(status)::schedule_status IS NOT NULL
            AND status NOT IN ('scheduled', 'fetching', 'delivering')
            AND sqlc.narg(status)::schedule_status != 'error'
        THEN sqlc.narg(status)::schedule_status
        ELSE status 
    END,
    updated_at = NOW()
WHERE job_id = sqlc.arg(job_id)
RETURNING job_id;

-- name: ActivateJob :one
UPDATE job_schedules
SET status = 'idle', updated_at = NOW()
WHERE job_id = sqlc.arg(job_id)
    AND status NOT IN ('scheduled', 'fetching', 'delivering')
RETURNING job_id;

-- name: DeactivateJob :one
UPDATE job_schedules
SET status = 'disabled', updated_at = NOW()
WHERE job_id = sqlc.arg(job_id)
    AND status NOT IN ('scheduled', 'fetching', 'delivering')
RETURNING job_id;

-- =========================
-- IO CONFIGS
-- =========================

-- name: CreateJobIOConfig :exec
INSERT INTO job_io_configs (
    job_id,
    kind,
    payload,
    headers,
    target_url,
    method,
    json_schema
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
);

-- name: PatchJobIOConfig :one
UPDATE job_io_configs
SET
    payload = CASE 
        WHEN sqlc.arg(set_payload)::bool THEN sqlc.narg(payload) 
        ELSE payload
    END,
    target_url = COALESCE(sqlc.narg(target_url), target_url),
    method = COALESCE(sqlc.narg(method), method),
    headers = CASE
        WHEN sqlc.arg(set_headers)::bool THEN sqlc.narg(headers)
        ELSE headers
    END,
    json_schema = CASE 
        WHEN sqlc.arg(set_json_schema)::bool THEN sqlc.narg(json_schema)
        ELSE json_schema
    END
WHERE job_id = sqlc.arg(job_id)
  AND kind = sqlc.arg(kind)::job_io_kind
RETURNING job_id;

-- name: ListJobIOConfigs :many
SELECT
    job_id,
    kind,
    payload,
    headers,
    json_schema,
    target_url,
    method
FROM job_io_configs
WHERE job_id = $1;
