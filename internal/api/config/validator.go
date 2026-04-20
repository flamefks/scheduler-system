package config

import (
	"time"
)

func ValidateCore(cfg *CoreConfig) (*CoreConfig, error) {
	if cfg.Service.ServiceName == "" {
		cfg.Service.ServiceName = "Api_Service"
	}

	if cfg.Service.Version == "" {
		cfg.Service.Version = "0.0.1"
	}

	if cfg.HTTP.Host == "" {
		cfg.HTTP.Host = "localhost"
	}

	if cfg.HTTP.Port == 0 {
		cfg.HTTP.Port = 8080
	}

	if cfg.HTTP.ReadTimeout == 0 {
		cfg.HTTP.ReadTimeout = 10 * time.Second
	}

	if cfg.HTTP.WriteTimeout == 0 {
		cfg.HTTP.WriteTimeout = 10 * time.Second
	}

	if cfg.HTTP.IdleTimeout == 0 {
		cfg.HTTP.IdleTimeout = 30 * time.Second
	}

	return cfg, nil

}
