package nats

import "github.com/nats-io/nats.go"

type Publisher struct {
	js nats.JetStreamContext
}

func NewPublisher(js nats.JetStreamContext) *Publisher {
	return &Publisher{js: js}
}

func (p *Publisher) Publish(subject string, data []byte) error {
	_, err := p.js.Publish(subject, data)
	return err
}
