package setup

import (
	"context"

	sharedData "github.com/flamefks/scheduler-system/internal/shared/data"
	"github.com/nats-io/nats.go/jetstream"
)

func EnsureStreams(ctx context.Context, js jetstream.JetStream) error {
	_, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name: sharedData.NatsStreamName,
		Subjects: []string{
			sharedData.JobsSubjectDeliver, sharedData.JobsSubjectFetcher,
		},
		Storage: jetstream.FileStorage,
	})
	return err
}
