ALTER TABLE job_schedules
DROP COLUMN scheduled_runs;

ALTER TABLE job_schedules
ADD COLUMN done_runs INT NOT NULL DEFAULT 0;
