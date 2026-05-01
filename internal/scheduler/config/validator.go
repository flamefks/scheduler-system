package config

import (
	"time"

	"github.com/nats-io/nats.go"
)

func ValidateCore(cfg *CoreConfig) (*CoreConfig, error) {
	if cfg.Service.ServiceName == "" {
		cfg.Service.ServiceName = "Scheduler_service"
	}

	if cfg.Service.Version == "" {
		cfg.Service.Version = "0.0.1"
	}

	if cfg.Nats.Url == "" {
		cfg.Nats.Url = nats.DefaultURL
	}

	if cfg.GetJobPollInterval == 0 {
		cfg.GetJobPollInterval = 500 * time.Millisecond
	}

	if cfg.BackgroundTasks.HungJobsMonitor.PollInterval == 0 {
		cfg.BackgroundTasks.HungJobsMonitor.PollInterval = 500 * time.Millisecond
	}

	if cfg.BackgroundTasks.HungJobsMonitor.JobDeathSecondsTimeout == 0 {
		cfg.BackgroundTasks.HungJobsMonitor.JobDeathSecondsTimeout = 900
	}

	if cfg.BackgroundTasks.DisableJobsMonitor.PollInterval == 0 {
		cfg.BackgroundTasks.DisableJobsMonitor.PollInterval = 500 * time.Millisecond
	}

	return cfg, nil

}
