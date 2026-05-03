package main

import (
	"context"
	"log"
	"os"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	natsSetup "github.com/flamefks/scheduler-system/internal/shared/queue/nats/setup"
)

func main() {
	url := os.Getenv("NATS_URL")

	ctx := context.Background()

	nc, err := nats.Connect(url)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Drain()

	js, err := jetstream.New(nc)
	if err != nil {
		log.Fatal(err)
	}

	if err := natsSetup.EnsureStreams(ctx, js); err != nil {
		log.Fatal(err)
	}

	log.Println("NATS streams successfully initialized")
}
