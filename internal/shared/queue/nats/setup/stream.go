package setup

import (
	"context"

	sharedData "github.com/flamefks/scheduler-system/internal/shared/data"
	"github.com/nats-io/nats.go/jetstream"
)

func EnsureStreams(ctx context.Context, js jetstream.JetStream) error {
	stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name: sharedData.NatsStreamName,
		Subjects: []string{
			sharedData.JobsSubjectDeliver, sharedData.JobsSubjectFetcher,
		},
		Storage: jetstream.FileStorage,
	})

	_, err = stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       sharedData.FetcherGroup,
		Name:          sharedData.FetcherGroup,
		FilterSubject: sharedData.JobsSubjectFetcher,
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return err
	}

	_, err = stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       sharedData.DeliverGroup,
		Name:          sharedData.DeliverGroup,
		FilterSubject: sharedData.JobsSubjectDeliver,
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return err
	}

	return err
}
