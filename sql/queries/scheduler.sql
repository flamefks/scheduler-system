-- name: ClaimNextJob :one
UPDATE job_schedules
SET
    status = 'running',
    scheduled_runs = scheduled_runs + 1,
    last_run_at = NOW(),
    next_run_at = CASE
        WHEN repeat_interval_sec > 0
            THEN NOW() + (repeat_interval_sec * INTERVAL '1 second')
        ELSE NULL
    END,
    updated_at = NOW()
WHERE job_id = (
    SELECT s.job_id
    FROM job_schedules s
    WHERE s.status = 'idle'
      AND s.next_run_at IS NOT NULL
      AND s.next_run_at <= NOW()
      AND (
            s.target_runs IS NULL
            OR s.scheduled_runs < s.target_runs
          )
    ORDER BY s.next_run_at
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING job_id;

-- name: ResetHungMessage :exec
UPDATE job_schedules
SET
    status = 'idle',
    updated_at = NOW()
WHERE status = 'active'
  AND NOW() - last_run_at > (sqlc.arg(timeout_seconds)::bigint * interval '1 second');

-- name: SwitchToDisabledIfNeed :exec
UPDATE job_schedules
SET
    status = 'disabled'
WHERE status = 'idle'
    AND scheduled_runs = target_runs;
