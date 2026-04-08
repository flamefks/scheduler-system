package config

type FileSettings struct {
	Path       string `yaml:"path"`
	MaxSizeMB  int    `yaml:"max_size_mb"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAgeDays int    `yaml:"max_age_days"`
	Compress   bool   `yaml:"compress"`
}

type LoggingConfig struct {
	Level  string       `yaml:"level"`
	Format string       `yaml:"format"`
	Output string       `yaml:"output"`
	File   FileSettings `yaml:"file"`
}
