package config

import (
	globalConf "github.com/flamefks/scheduler-system/internal/config"
)

func LoadCoreConfig(path string) (*CoreConfig, error) {
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
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
