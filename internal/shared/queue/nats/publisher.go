package nats

import (
	"context"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Publisher struct {
	js jetstream.JetStream
}

func NewPublisher(js jetstream.JetStream) *Publisher {
	return &Publisher{js: js}
}

func (p *Publisher) Publish(
	ctx context.Context,
	subject string,
	data []byte,
	headers map[string]string,
) error {
	msg := &nats.Msg{
		Subject: subject,
		Data:    data,
		Header:  nats.Header{},
	}

	for k, v := range headers {
		msg.Header.Set(k, v)
	}

	_, err := p.js.PublishMsg(ctx, msg)
	return err
}
