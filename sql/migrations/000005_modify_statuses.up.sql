ALTER TABLE job_schedules ALTER COLUMN status DROP DEFAULT;

CREATE TYPE schedule_status_new AS ENUM (
    'idle',
    'scheduled',
    'fetching',
    'delivering',
    'error',
    'disabled'
);

ALTER TABLE job_schedules
ALTER COLUMN status TYPE schedule_status_new
USING (
    CASE status::text
        WHEN 'running' THEN 'idle'
        ELSE status::text
    END
)::schedule_status_new;

DROP TYPE schedule_status;

ALTER TYPE schedule_status_new RENAME TO schedule_status;

ALTER TABLE job_schedules ALTER COLUMN status SET DEFAULT 'idle';

-- Change scheduled_runs --> done_runs
ALTER TABLE job_schedules
ADD COLUMN done_runs INT NOT NULL DEFAULT 0;

UPDATE job_schedules SET done_runs = scheduled_runs;

ALTER TABLE job_schedules
DROP COLUMN scheduled_runs;

-- Remove constraint target_runs
ALTER TABLE job_schedules
DROP CONSTRAINT IF EXISTS job_schedules_target_runs_check;
