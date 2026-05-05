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
	BackgroundTasks    BackgroundTasksSection `yaml:"background_tasks" json:"background_tasks"`
	GetJobPollInterval time.Duration          `yaml:"get_job_poll_interval" json:"poll_interval"`
}

type BackgroundTasksSection struct {
	HungJobsMonitor    HungJobsMonitorSection    `yaml:"hung_jobs_monitor" json:"hung_jobs_monitor"`
	DisableJobsMonitor DisableJobsMonitorSection `yaml:"disable_jobs_monitor" json:"disable_jobs_monitor"`
}

type HungJobsMonitorSection struct {
	PollInterval           time.Duration `yaml:"poll_interval" json:"poll_interval"`
	JobDeathSecondsTimeout int           `yaml:"job_death_seconds_timeout" json:"job_death_seconds_timeout"`
}

type DisableJobsMonitorSection struct {
	PollInterval time.Duration `yaml:"poll_interval" json:"poll_interval"`
}
