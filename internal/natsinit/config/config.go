package config

import (
	"sync"

	globalConf "github.com/flamefks/scheduler-system/internal/config"
	natsSetup "github.com/flamefks/scheduler-system/internal/shared/queue/nats/setup"
)

var (
	instance *CoreConfig
	once     sync.Once
	loadErr  error
)

type CoreConfig struct {
	Nats struct {
		Url string `yaml:"url" json:"url"`
	} `yaml:"nats" json:"nats"`
	JetStream natsSetup.Config `yaml:"jetstream" json:"jetstream"`
}

func LoadAppConfig(path string) (*CoreConfig, error) {
	once.Do(func() {
		instance, loadErr = loadCoreConfig(path)
	})
	if loadErr != nil {
		return nil, loadErr
	}
	return instance, nil
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

	cfg, err = ValidateCore(cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
