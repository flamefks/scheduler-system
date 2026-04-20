package config

import (
	"time"

	generalConf "github.com/flamefks/scheduler-system/internal/config"
)

type CoreConfig struct {
	Service generalConf.ServiceSection `yaml:"service" json:"service"`
	HTTP    HttpSection                `yaml:"http" json:"http"`
	// HttpHandlerTimers HttpHandlerTimers            `yaml:"timers" json:"http_handler_timers"`
	Postgres *generalConf.PostgresSection `yaml:"database" json:"database"`
}

// type ContextTimers struct {
// 	DbWriteTimeout time.Duration
// 	DbReadTimeout  time.Duration
// }

type HttpSection struct {
	Host            string        `yaml:"host" json:"host"`
	Port            int           `yaml:"port" json:"port"`
	ReadTimeout     time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout" json:"write_timeout"`
	IdleTimeout     time.Duration `yaml:"idle_timeout" json:"idle_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" json:"shutdown_timeout"`
}
