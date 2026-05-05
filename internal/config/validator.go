package config

import (
	"fmt"
	"time"
)

func ValidateLogging(cfg *LoggingConfig) (*LoggingConfig, error) {
	if cfg.Logger != nil {
		cfg.Output = "file"

		if cfg.Logger.Path == "" {
			return nil, fmt.Errorf("logging config: logger.path is required when output=file")
		}

		if cfg.Logger.MaxSizeMB == 0 {
			cfg.Logger.MaxSizeMB = 5
		}

		if cfg.Logger.MaxBackups == 0 {
			cfg.Logger.MaxBackups = 5
		}

		if cfg.Logger.MaxAgeDays == 0 {
			cfg.Logger.MaxAgeDays = 30
		}

	} else {
		cfg.Output = "stdout"
	}

	return cfg, nil
}

func ValidateDbSection(cfg *PostgresSection) (*PostgresSection, error) {
	if cfg == nil || cfg.DSN == "" {
		return nil, fmt.Errorf("no database source name found")
	}

	if cfg.MaxConns == 0 {
		cfg.MaxConns = 10
	}

	if cfg.MinConns == 0 {
		cfg.MinConns = 2
	}

	if cfg.MaxConnLifetime == 0 {
		cfg.MaxConnLifetime = 1 * time.Hour
	}

	if cfg.MaxConnIdleTime == 0 {
		cfg.MaxConnIdleTime = 30 * time.Minute
	}

	if cfg.HealthCheckPeriod == 0 {
		cfg.HealthCheckPeriod = 30 * time.Second
	}
	return cfg, nil
}

func ValidateHttpRetrySection(cfg *HttpRetryPolicySection) (*HttpRetryPolicySection, error) {
	if cfg.MaxAttempts == 0 {
		cfg.MaxAttempts = 3
	}

	if cfg.BaseDelay == 0 {
		cfg.BaseDelay = time.Second * 5
	}

	if cfg.MaxDelay == 0 || cfg.MaxDelay < cfg.BaseDelay {
		cfg.MaxDelay = cfg.BaseDelay + 1
	}

	if cfg.Backoff != "fixed" && cfg.Backoff == "exponential" {
		cfg.Backoff = "fixed"
	}

	return cfg, nil

}
