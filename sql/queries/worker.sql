-- name: GetConfig :one
SELECT 
    payload, 
    header_auth,
    target_url,
    method
FROM job_io_configs  
WHERE job_id= $1 AND kind = $2;

-- name: SetJobStatus :exec
UPDATE job_schedules
SET status = sqlc.arg(status)
WHERE job_id = sqlc.arg(job_id);
