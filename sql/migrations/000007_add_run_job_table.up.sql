-- Removing old columns
ALTER TABLE job_schedules
DROP COLUMN last_run_taken_at;

ALTER TABLE job_schedules
DROP COLUMN status;

ALTER TABLE job_schedules
ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT TRUE;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Creating job_run table
CREATE TABLE job_runs (
    id UUID PRIMARY KEY,
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    status schedule_status NOT NULL DEFAULT 'scheduled',
    scheduled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    fetch_started_at TIMESTAMPTZ NULL,
    deliver_started_at TIMESTAMPTZ NULL,
    ended_at TIMESTAMPTZ NULL
);

CREATE INDEX idx_job_runs_job_status
    ON job_runs (job_id, status);

CREATE INDEX idx_job_runs_status_scheduled_at
    ON job_runs (status, scheduled_at);

