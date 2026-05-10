ALTER TABLE job_schedules ALTER COLUMN status DROP DEFAULT;

CREATE TYPE schedule_status_old AS ENUM (
    'idle',
    'running',
    'error',
    'disabled'
);

ALTER TABLE job_schedules
ALTER COLUMN status TYPE schedule_status_old
USING (
    CASE status::text
        WHEN 'scheduled' THEN 'idle'
        WHEN 'fetching' THEN 'idle'
        WHEN 'delivering' THEN 'idle'
        ELSE status::text
    END
)::schedule_status_old;

DROP TYPE schedule_status;

ALTER TYPE schedule_status_old RENAME TO schedule_status;

ALTER TABLE job_schedules ALTER COLUMN status SET DEFAULT 'idle';
