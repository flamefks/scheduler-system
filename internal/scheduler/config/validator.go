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

	if cfg.JobPollInterval == 0 {
		cfg.JobPollInterval = 500 * time.Millisecond
	}

	if cfg.JobDeathSecondsTimeout == 0 {
		cfg.JobDeathSecondsTimeout = 900
	}
	return cfg, nil

}
