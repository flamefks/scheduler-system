package config

import (
	"time"

	"github.com/nats-io/nats.go"
)

func ValidateCore(cfg *CoreConfig) (*CoreConfig, error) {
	if cfg.Nats.Url == "" {
		cfg.Nats.Url = nats.DefaultURL
	}

	if cfg.JetStream.Stream.Storage == "" {
		cfg.JetStream.Stream.Storage = "file"
	}

	if cfg.JetStream.Stream.Retention == "" {
		cfg.JetStream.Stream.Retention = "workqueue"
	}

	if cfg.JetStream.Consumer.AckWait == 0 {
		cfg.JetStream.Consumer.AckWait = 2 * time.Minute
	}

	if cfg.JetStream.Consumer.MaxDeliver == 0 {
		cfg.JetStream.Consumer.MaxDeliver = 5
	}

	if cfg.JetStream.Consumer.MaxAckPending == 0 {
		cfg.JetStream.Consumer.MaxAckPending = 128
	}

	if len(cfg.JetStream.Consumer.BackOff) == 0 {
		cfg.JetStream.Consumer.BackOff = []time.Duration{
			5 * time.Second,
			30 * time.Second,
			2 * time.Minute,
			5 * time.Minute,
		}
	}

	return cfg, nil
}
