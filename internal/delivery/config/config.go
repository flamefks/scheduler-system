package config

import (
	generalConf "github.com/flamefks/scheduler-system/internal/config"
)

type CoreConfig struct {
	Service  generalConf.ServiceSection
	Postgres *generalConf.PostgresSection
	Nats     struct {
		Url string
	}
}
