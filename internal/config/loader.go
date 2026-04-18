package config

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func LoadYAML[T any](path string) (*T, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg T
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)

	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config %s: %w", path, err)
	}

	return &cfg, nil
}

func LoadLogging(path string) (*LoggingConfig, error) {
	cfg, err := LoadYAML[LoggingConfig](path)
	if err != nil {
		return nil, err
	}
	cfg, err = ValidateLogging(cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
