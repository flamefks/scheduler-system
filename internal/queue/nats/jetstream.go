package nats

import (
	"context"
	"log"
	"time"

	shared "github.com/flamefks/scheduler-system/internal/shared"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func NewJetStream(backgrCtx context.Context, nc *nats.Conn) jetstream.JetStream {
	ctx, cancel := context.WithTimeout(backgrCtx, 10*time.Second)
	defer cancel()

	js, err := jetstream.New(nc)
	if err != nil {
		log.Fatal(err)
	}

	_, err = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{Name: "jobs", Subjects: []string{
		shared.JobsSubjectDeliver, shared.JobsSubjectFetcher}})
	if err != nil {
		log.Fatal(err)
	}

	return js
}
