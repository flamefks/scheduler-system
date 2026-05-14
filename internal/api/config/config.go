package config

import (
	"sync"
	"time"

	generalConf "github.com/flamefks/scheduler-system/internal/config"
	globalConf "github.com/flamefks/scheduler-system/internal/config"
)

var (
	instance *CoreConfig
	once     sync.Once
	loadErr  error
)

type CoreConfig struct {
	Service     generalConf.ServiceSection   `yaml:"service" json:"service"`
	HTTP        HttpSection                  `yaml:"http" json:"http"`
	Postgres    *generalConf.PostgresSection `yaml:"database" json:"database"`
	OtelSection generalConf.OtelSection      `yaml:"otel" json:"otel"`
}

type HttpSection struct {
	Host            string        `yaml:"host" json:"host"`
	Port            int           `yaml:"port" json:"port"`
	ReadTimeout     time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout" json:"write_timeout"`
	IdleTimeout     time.Duration `yaml:"idle_timeout" json:"idle_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" json:"shutdown_timeout"`
}

func LoadAppConfig(path string) (*CoreConfig, error) {
	once.Do(func() {
		instance, loadErr = loadCoreConfig(path)
	})
	if loadErr != nil {
		return nil, loadErr
	} else {
		return instance, nil
	}
}

func GetCoreConfig() *CoreConfig {
	if instance == nil {
		panic("config not initialized before")
	}
	return instance
}

func loadCoreConfig(path string) (*CoreConfig, error) {
	cfg, err := globalConf.LoadYAML[CoreConfig](path)
	if err != nil {
		return nil, err
	}

	dbSection, err := globalConf.ValidateDbSection(cfg.Postgres)
	if err != nil {
		return nil, err
	}
	cfg.Postgres = dbSection
	cfg, err = ValidateCore(cfg)

	otelSection, err := globalConf.ValidateOtelSection(&cfg.OtelSection)
	if err != nil {
		return nil, err
	}
	cfg.OtelSection = *otelSection

	if err != nil {
		return nil, err
	}
	return cfg, nil
}
