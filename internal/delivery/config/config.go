package config

import (
	"sync"

	globalConf "github.com/flamefks/scheduler-system/internal/config"
)

var (
	instance *CoreConfig
	once     sync.Once
	loadErr  error
)

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

type CoreConfig struct {
	Service   globalConf.ServiceSection         `yaml:"service" json:"service"`
	Postgres  *globalConf.PostgresSection       `yaml:"database" json:"database"`
	HttpRetry globalConf.HttpRetryPolicySection `yaml:"http_retry" json:"http_retry"`
	Nats      struct {
		Url string `yaml:"url" json:"url"`
	} `yaml:"nats" json:"nats"`
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

	httpRetrySection, err := globalConf.ValidateHttpRetrySection(&cfg.HttpRetry)
	if err != nil {
		return nil, err
	}

	cfg.Postgres = dbSection
	cfg.HttpRetry = *httpRetrySection
	cfg, err = ValidateCore(cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
