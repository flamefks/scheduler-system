DROP INDEX IF EXISTS idx_job_runs_status_scheduled_at;
DROP INDEX IF EXISTS idx_job_runs_job_status;

DROP TABLE IF EXISTS job_runs;

ALTER TABLE job_schedules
DROP COLUMN is_active;

ALTER TABLE job_schedules
ADD COLUMN status schedule_status NOT NULL DEFAULT 'idle';

ALTER TABLE job_schedules
ADD COLUMN last_run_taken_at TIMESTAMPTZ NULL;
