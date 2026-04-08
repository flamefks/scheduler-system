package main

import (
	"log"

	"github.com/nats-io/nats.go"

	natsqueue "github.com/flamefks/scheduler-system/internal/queue/nats"

	"github.com/flamefks/scheduler-system/internal/fetcher/service"
)

func main() {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatal(err)
	}

	js, err := nc.JetStream()
	if err != nil {
		log.Fatal(err)
	}

	fetcher := service.NewFetcherService()

	consumer := natsqueue.NewConsumer(js, "jobs.fetch")

	log.Println("Fetcher worker started...")

	consumer.Consume(func(msg []byte) error {
		return fetcher.Handle(msg)
	})
}
