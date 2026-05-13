ALTER TABLE job_schedules
DROP COLUMN last_run_taken_at;

ALTER TABLE job_schedules
RENAME COLUMN last_scheduled_at TO last_run_at;
