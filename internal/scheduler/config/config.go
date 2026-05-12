package config

import (
	"time"

	generalConf "github.com/flamefks/scheduler-system/internal/config"
)

type CoreConfig struct {
	Service  generalConf.ServiceSection   `yaml:"service" json:"service"`
	Postgres *generalConf.PostgresSection `yaml:"database" json:"database"`
	Nats     struct {
		Url string `yaml:"url" json:"url"`
	} `yaml:"nats" json:"nats"`
	Tasks TasksSection `yaml:"background_tasks" json:"background_tasks"`
}

type TasksSection struct {
	HungJobsMonitor    HungJobsMonitorSection    `yaml:"hung_jobs_monitor" json:"hung_jobs_monitor"`
	DisableJobsMonitor DisableJobsMonitorSection `yaml:"disable_jobs_monitor" json:"disable_jobs_monitor"`
	GetJobSettings     GetJobSection             `yaml:"get_job" json:"get_job"`
}

type GetJobSection struct {
	PollInterval           time.Duration `yaml:"poll_interval" json:"poll_interval"`
	JobsBatchSize          int           `yaml:"jobs_batch_size" json:"jobs_batch_size"`
	MaxParallerNatsPushers int           `yaml:"max_parallel_pushers" json:"max_parallel_pushers"`
}

type HungJobsMonitorSection struct {
	PollInterval           time.Duration `yaml:"poll_interval" json:"poll_interval"`
	ScheduleTimeoutSeconds int           `yaml:"schedule_timeout_seconds" json:"schedule_timeout_seconds"`
	ProcTimeoutSeconds     int           `yaml:"proc_timeout_seconds" json:"proc_timeout_seconds"`
}

type DisableJobsMonitorSection struct {
	PollInterval time.Duration `yaml:"poll_interval" json:"poll_interval"`
}
