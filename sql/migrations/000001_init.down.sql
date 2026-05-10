DROP INDEX IF EXISTS idx_job_schedules_status_next_run;
DROP INDEX IF EXISTS idx_job_schedules_status;

DROP TABLE IF EXISTS job_io_configs;
DROP TABLE IF EXISTS job_schedules;
DROP TABLE IF EXISTS jobs;

DROP TYPE IF EXISTS schedule_status;
DROP TYPE IF EXISTS job_io_kind;
