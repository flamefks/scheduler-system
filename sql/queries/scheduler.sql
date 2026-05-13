-- name: ClaimNextJobs :many
WITH picked AS (
    SELECT s.job_id
    FROM job_schedules s
    WHERE s.status = 'idle'
      AND s.next_run_at IS NOT NULL
      AND s.next_run_at <= NOW()
      AND (
            s.target_runs = 0
            OR s.done_runs < s.target_runs
          )
    ORDER BY s.next_run_at
    FOR UPDATE SKIP LOCKED
    LIMIT sqlc.arg(batch_size)::int
)
UPDATE job_schedules s
SET
    status = 'scheduled',
    last_scheduled_at = NOW(),
    next_run_at = CASE
        WHEN target_runs != 0 AND done_runs + 1 >= target_runs
            THEN NULL
        WHEN s.repeat_interval_sec > 0
            THEN NOW() + (s.repeat_interval_sec * INTERVAL '1 second')
        ELSE NOW()
    END,
    updated_at = NOW()
FROM picked
WHERE s.job_id = picked.job_id
RETURNING s.job_id;

-- name: ResetHungMessage :exec
UPDATE job_schedules
SET
    status = 'idle',
    updated_at = NOW()
WHERE (status IN ('fetching', 'delivering')
  AND NOW() - COALESCE(last_run_taken_at, last_scheduled_at) > (sqlc.arg(proc_timeout_seconds)::bigint * interval '1 second'))
OR (status = 'scheduled'
  AND NOW() - last_scheduled_at > (sqlc.arg(schedule_timeout_seconds)::bigint * interval '1 second'));

-- name: SwitchToDisabledIfNeed :exec
UPDATE job_schedules
SET
    status = 'disabled'
WHERE status = 'idle'
    AND done_runs >= target_runs AND target_runs != 0;
