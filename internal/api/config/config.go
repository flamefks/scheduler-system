package config

import (
	"time"

	generalConf "github.com/flamefks/scheduler-system/internal/config"
)

type CoreConfig struct {
	Service  generalConf.ServiceSection
	HTTP     HttpSection
	Postgres *generalConf.PostgresSection
}

type HttpSection struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}
