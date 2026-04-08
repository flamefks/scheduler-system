package nats

import (
	"log"

	"github.com/nats-io/nats.go"
)

type Consumer struct {
	js      nats.JetStreamContext
	subject string
}

func NewConsumer(js nats.JetStreamContext, subject string) *Consumer {
	return &Consumer{
		js:      js,
		subject: subject,
	}
}

func (c *Consumer) Consume(handler func([]byte) error) {
	_, err := c.js.Subscribe(c.subject, func(msg *nats.Msg) {
		if err := handler(msg.Data); err != nil {
			log.Println("handler error:", err)
			return
		}

		msg.Ack()
	})

	if err != nil {
		log.Fatal(err)
	}

	select {}
}
