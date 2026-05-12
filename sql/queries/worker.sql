-- name: GetConfig :one
SELECT 
    payload, 
    headers,
    target_url,
    method,
    json_schema
FROM job_io_configs  
WHERE job_id= $1 AND kind = $2;

-- name: SetJobStatus :exec
UPDATE job_schedules
SET
    status = sqlc.arg(status),
    done_runs = CASE
        WHEN sqlc.arg(status)::schedule_status IN ('idle', 'error')
            THEN done_runs + 1
        ELSE done_runs
    END,
    updated_at = NOW()
WHERE job_id = sqlc.arg(job_id);
