package setup

import (
	"context"
	"fmt"
	"time"

	sharedData "github.com/flamefks/scheduler-system/internal/shared/data"
	"github.com/nats-io/nats.go/jetstream"
)

type StreamSettings struct {
	Storage   string
	Retention string
}

type ConsumerSettings struct {
	AckWait       time.Duration
	MaxDeliver    int
	BackOff       []time.Duration
	MaxAckPending int
}

type Config struct {
	Stream   StreamSettings
	Consumer ConsumerSettings
}

func EnsureStreams(ctx context.Context, js jetstream.JetStream, cfg Config) error {
	storage, err := parseStorage(cfg.Stream.Storage)
	if err != nil {
		return err
	}
	retention, err := parseRetention(cfg.Stream.Retention)
	if err != nil {
		return err
	}

	stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name: sharedData.NatsStreamName,
		Subjects: []string{
			sharedData.JobsSubjectDeliver, sharedData.JobsSubjectFetcher,
		},
		Storage:   storage,
		Retention: retention,
	})
	if err != nil {
		return err
	}

	_, err = stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       sharedData.FetcherGroup,
		Name:          sharedData.FetcherGroup,
		FilterSubject: sharedData.JobsSubjectFetcher,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       cfg.Consumer.AckWait,
		MaxDeliver:    cfg.Consumer.MaxDeliver,
		BackOff:       cfg.Consumer.BackOff,
		MaxAckPending: cfg.Consumer.MaxAckPending,
	})
	if err != nil {
		return err
	}

	_, err = stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       sharedData.DeliverGroup,
		Name:          sharedData.DeliverGroup,
		FilterSubject: sharedData.JobsSubjectDeliver,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       cfg.Consumer.AckWait,
		MaxDeliver:    cfg.Consumer.MaxDeliver,
		BackOff:       cfg.Consumer.BackOff,
		MaxAckPending: cfg.Consumer.MaxAckPending,
	})
	if err != nil {
		return err
	}

	return err
}

func parseStorage(raw string) (jetstream.StorageType, error) {
	switch raw {
	case "", "file":
		return jetstream.FileStorage, nil
	case "memory":
		return jetstream.MemoryStorage, nil
	default:
		return 0, fmt.Errorf("unsupported nats stream storage: %s", raw)
	}
}

func parseRetention(raw string) (jetstream.RetentionPolicy, error) {
	switch raw {
	case "", "workqueue":
		return jetstream.WorkQueuePolicy, nil
	case "limits":
		return jetstream.LimitsPolicy, nil
	case "interest":
		return jetstream.InterestPolicy, nil
	default:
		return 0, fmt.Errorf("unsupported nats stream retention: %s", raw)
	}
}
