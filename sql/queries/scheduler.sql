-- name: ClaimNextJobs :many
WITH picked AS (
    SELECT s.job_id, gen_random_uuid() AS run_id
    FROM job_schedules s
    WHERE s.is_active = TRUE
      AND s.next_run_at IS NOT NULL
      AND s.next_run_at <= NOW()
      AND (
            s.target_runs = 0
            OR s.done_runs < s.target_runs
          )
      AND NOT EXISTS (
            SELECT 1
            FROM job_runs r
            WHERE r.job_id = s.job_id
              AND r.status IN ('scheduled', 'fetching', 'delivering')
          )
    ORDER BY s.next_run_at
    FOR UPDATE SKIP LOCKED
    LIMIT sqlc.arg(batch_size)::int
),
updated AS (
UPDATE job_schedules s
SET
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
RETURNING s.job_id, picked.run_id
),
inserted_runs AS (
    INSERT INTO job_runs (
        id,
        job_id,
        status,
        scheduled_at
    )
    SELECT
        run_id,
        job_id,
        'scheduled',
        NOW()
    FROM updated
    RETURNING job_id, id
)
SELECT
    job_id,
    id AS run_id
FROM inserted_runs;

-- name: ResetHungMessage :execrows
UPDATE job_runs
SET
    status = 'error',
    ended_at = NOW()
WHERE (status IN ('fetching', 'delivering')
  AND NOW() - COALESCE(deliver_started_at, fetch_started_at, scheduled_at) > (sqlc.arg(proc_timeout_seconds)::bigint * interval '1 second'))
OR (status = 'scheduled'
  AND NOW() - scheduled_at > (sqlc.arg(schedule_timeout_seconds)::bigint * interval '1 second'));

-- name: SwitchToDisabledIfNeed :execrows
UPDATE job_schedules
SET
    is_active = FALSE
WHERE is_active = TRUE
    AND done_runs >= target_runs AND target_runs != 0;
