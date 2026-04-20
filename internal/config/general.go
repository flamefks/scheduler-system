package config

import "time"

type ServiceSection struct {
	ServiceName string
	Version     string
}

type PostgresSection struct {
	DSN               string        `yaml:"url" json:"url"`
	MaxConns          int32         `yaml:"max_connections" json:"max_connections"`
	MinConns          int32         `yaml:"min_connections" json:"min_connections"`
	MaxConnLifetime   time.Duration `yaml:"max_connection_lifetime" json:"max_connection_lifetime"`
	MaxConnIdleTime   time.Duration `yaml:"min_connection_lifetime" json:"min_connection_lifetime"`
	HealthCheckPeriod time.Duration `yaml:"healthcheck_period" json:"healthcheck_period"`
}

type FileSettings struct {
	Path       string `yaml:"path" json:"path"`
	MaxSizeMB  int    `yaml:"max_size_mb" json:"max_size_mb"`
	MaxBackups int    `yaml:"max_backups" json:"max_backups"`
	MaxAgeDays int    `yaml:"max_age_days" json:"max_age_days"`
	Compress   bool   `yaml:"compress" json:"compress"`
}

type LoggingConfig struct {
	Level  string `yaml:"level" json:"level"`
	Format string `yaml:"format" json:"format"`
	Output string `yaml:"output" json:"output"`

	Logger *FileSettings `yaml:"logger" json:"logger"`
}

type HttpRetryPolicySection struct {
}
