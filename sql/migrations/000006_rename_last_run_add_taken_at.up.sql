ALTER TABLE job_schedules
RENAME COLUMN last_run_at TO last_scheduled_at;

ALTER TABLE job_schedules
ADD COLUMN last_run_taken_at TIMESTAMPTZ NULL;
