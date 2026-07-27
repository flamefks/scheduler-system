-- name: GetConfig :one
SELECT 
    payload, 
    headers,
    target_url,
    method,
    json_schema
FROM job_io_configs  
WHERE job_id= $1 AND kind = $2;

-- name: SetJobRunStatus :one
WITH updated_run AS (
UPDATE job_runs
SET
    status = sqlc.arg(status),
    fetch_started_at = CASE
        WHEN sqlc.arg(status)::schedule_status = 'fetching'
            THEN NOW()
        ELSE fetch_started_at
    END,
    deliver_started_at = CASE
        WHEN sqlc.arg(status)::schedule_status = 'delivering'
            THEN NOW()
        ELSE deliver_started_at
    END,
    ended_at = CASE
        WHEN sqlc.arg(status)::schedule_status IN ('idle', 'error')
            THEN NOW()
        ELSE ended_at
    END
WHERE id = sqlc.arg(run_id)
  AND job_id = sqlc.arg(job_id)
  AND (
        (sqlc.arg(status)::schedule_status = 'fetching' AND status = 'scheduled')
        OR (sqlc.arg(status)::schedule_status = 'delivering' AND status = 'fetching')
        OR (sqlc.arg(status)::schedule_status IN ('idle', 'error') AND status IN ('scheduled', 'fetching', 'delivering'))
      )
RETURNING job_id
)
UPDATE job_schedules s
SET
    done_runs = CASE
        WHEN sqlc.arg(status)::schedule_status IN ('idle', 'error')
            THEN done_runs + 1
        ELSE done_runs
    END,
    updated_at = NOW()
FROM updated_run ur
WHERE s.job_id = ur.job_id
RETURNING s.job_id;
