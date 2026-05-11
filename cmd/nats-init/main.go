package main

import (
	"context"
	"log"

	coreConf "github.com/flamefks/scheduler-system/internal/natsinit/config"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	natsSetup "github.com/flamefks/scheduler-system/internal/shared/queue/nats/setup"
)

func main() {
	ctx := context.Background()

	coreCfg, err := coreConf.LoadAppConfig("config/core.yml")
	if err != nil {
		log.Fatal(err)
	}

	nc, err := nats.Connect(coreCfg.Nats.Url)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Drain()

	js, err := jetstream.New(nc)
	if err != nil {
		log.Fatal(err)
	}

	if err := natsSetup.EnsureStreams(ctx, js, coreCfg.JetStream); err != nil {
		log.Fatal(err)
	}

	log.Println("NATS streams successfully initialized")
}
