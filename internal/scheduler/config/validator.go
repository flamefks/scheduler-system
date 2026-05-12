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

	if cfg.Tasks.GetJobSettings.PollInterval == 0 {
		cfg.Tasks.GetJobSettings.PollInterval = 500 * time.Millisecond
	}

	if cfg.Tasks.GetJobSettings.JobsBatchSize == 0 {
		cfg.Tasks.GetJobSettings.JobsBatchSize = 100
	}

	if cfg.Tasks.GetJobSettings.MaxParallerNatsPushers == 0 {
		cfg.Tasks.GetJobSettings.MaxParallerNatsPushers = 256
	}

	if cfg.Tasks.HungJobsMonitor.PollInterval == 0 {
		cfg.Tasks.HungJobsMonitor.PollInterval = 500 * time.Millisecond
	}

	if cfg.Tasks.HungJobsMonitor.ScheduleTimeoutSeconds == 0 {
		cfg.Tasks.HungJobsMonitor.ScheduleTimeoutSeconds = 15
	}

	if cfg.Tasks.HungJobsMonitor.ProcTimeoutSeconds == 0 {
		cfg.Tasks.HungJobsMonitor.ProcTimeoutSeconds = 900
	}

	if cfg.Tasks.DisableJobsMonitor.PollInterval == 0 {
		cfg.Tasks.DisableJobsMonitor.PollInterval = 500 * time.Millisecond
	}

	return cfg, nil

}
