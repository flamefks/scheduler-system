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
	JobDeathSecondsTimeout int64         `yaml:"job_death_seconds_timeout" json:"job_death_seconds_timeout"`
	JobPollInterval        time.Duration `yaml:"job_poll_interval" json:"job_poll_interval"`
}
