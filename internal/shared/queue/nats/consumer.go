package nats

import (
	"context"
	"fmt"
	"log"
	"time"

	sharedData "github.com/flamefks/scheduler-system/internal/shared/data"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Consumer struct {
	js      jetstream.JetStream
	subject string
}

func NewConsumer(js jetstream.JetStream, subject string) *Consumer {
	return &Consumer{
		js:      js,
		subject: subject,
	}
}

func (c *Consumer) Consume(appCtx context.Context, handler func(context.Context, []byte, nats.Header) error,
	errHandler func(context.Context, []byte, nats.Header) error, groupName string) error {

	initCtx, cancel := context.WithTimeout(appCtx, 5*time.Second)
	defer cancel()
	stream, err := c.js.Stream(initCtx, sharedData.NatsStreamName)
	if err != nil {
		log.Printf("worker: stream error: %v", err)
		return fmt.Errorf("worker: stream error: %v", err)
	}

	consumer, err := stream.Consumer(initCtx, groupName)
	if err != nil {
		return fmt.Errorf("worker: consumer error: %v", err)
	}

	cc, err := consumer.Consume(func(msg jetstream.Msg) {
		msgCtx, cancel := context.WithTimeout(appCtx, 2*time.Hour)
		defer cancel()

		binData := msg.Data()
		header := msg.Headers()
		err := handler(msgCtx, binData, header)
		if err != nil {
			errCtx, cancelErr := context.WithTimeout(appCtx, 10*time.Minute)
			defer cancelErr()

			if hErr := errHandler(errCtx, binData, header); hErr != nil {
				return
			}
		}

		msg.Ack()
	})

	if err != nil {
		return fmt.Errorf("start consume: %w", err)
	}
	defer cc.Stop()

	<-appCtx.Done()
	return nil
}
