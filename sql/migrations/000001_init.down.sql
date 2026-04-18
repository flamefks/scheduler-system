DROP INDEX IF EXISTS idx_job_runs_planned_at;
DROP INDEX IF EXISTS idx_job_runs_status;
DROP INDEX IF EXISTS idx_job_runs_job_id;

DROP INDEX IF EXISTS idx_job_schedules_repeat_interval_sec;
DROP INDEX IF EXISTS idx_job_schedules_start_at;

DROP INDEX IF EXISTS idx_jobs_status;

DROP TABLE IF EXISTS job_runs;
DROP TABLE IF EXISTS job_schedules;
DROP TABLE IF EXISTS jobs;

DROP TYPE IF EXISTS job_run_status;
DROP TYPE IF EXISTS job_status;