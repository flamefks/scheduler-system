-- TYPES

CREATE TYPE job_io_kind AS ENUM ('fetcher', 'deliver');

CREATE TYPE schedule_status AS ENUM ('idle', 'running', 'error', 'disabled');


-- JOBS (static definition)

CREATE TABLE jobs (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


-- SCHEDULES (runtime + planning)

CREATE TABLE job_schedules (
    job_id UUID PRIMARY KEY REFERENCES jobs(id) ON DELETE CASCADE,

    status schedule_status NOT NULL DEFAULT 'idle',

    repeat_interval_sec INT NOT NULL DEFAULT 0,
    done_runs INT NOT NULL DEFAULT 0,
    target_runs INT NOT NULL DEFAULT 0,

    last_run_at TIMESTAMPTZ NULL,
    next_run_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT job_schedules_repeat_interval_check
        CHECK (repeat_interval_sec >= 0),

    CONSTRAINT job_schedules_target_runs_check
        CHECK (target_runs IS NULL OR target_runs > 0)
);


-- IO CONFIGS (fetcher / deliver)

CREATE TABLE job_io_configs (
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    kind job_io_kind NOT NULL,

    payload JSONB NOT NULL,
    headers JSONB NULL,
    target_url TEXT NOT NULL,
    method TEXT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (job_id, kind)
);


-- CREATE TABLE job_retry_policies (
--     job_id UUID PRIMARY KEY REFERENCES jobs(id) ON DELETE CASCADE,

--     max_attempts INT NOT NULL DEFAULT 3,
--     backoff_type TEXT NOT NULL DEFAULT 'fixed', -- fixed, exponential
--     base_delay_sec INT NOT NULL DEFAULT 5,
--     max_delay_sec INT NULL,
--     retry_on_http_5xx BOOLEAN NOT NULL DEFAULT TRUE,
--     retry_on_timeout BOOLEAN NOT NULL DEFAULT TRUE,
--     retry_on_network_error BOOLEAN NOT NULL DEFAULT TRUE,

--     created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
--     updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

--     CHECK (max_attempts >= 0),
--     CHECK (base_delay_sec >= 0),
--     CHECK (max_delay_sec IS NULL OR max_delay_sec >= base_delay_sec)
-- );

-- INDEXES

CREATE INDEX idx_job_schedules_status_next_run
    ON job_schedules (status, next_run_at);

CREATE INDEX idx_job_schedules_status
    ON job_schedules (status);