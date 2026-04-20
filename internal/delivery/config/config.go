package config

import (
	generalConf "github.com/flamefks/scheduler-system/internal/config"
)

type CoreConfig struct {
	Service  generalConf.ServiceSection   `yaml:"service" json:"service"`
	Postgres *generalConf.PostgresSection `yaml:"database" json:"database"`
	Nats     struct {
		Url string `yaml:"url" json:"url"`
	} `yaml:"nats" json:"nats"`
}
