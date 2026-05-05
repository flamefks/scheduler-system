ALTER TABLE job_schedules
ADD COLUMN scheduled_runs INT NOT NULL DEFAULT 0;

ALTER TABLE job_schedules
DROP COLUMN done_runs;
