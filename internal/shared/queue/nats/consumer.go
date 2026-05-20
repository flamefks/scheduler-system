package nats

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	sharedData "github.com/flamefks/scheduler-system/internal/shared/data"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

var (
	NakError  = errors.New("nak error")
	TermError = errors.New("term error")
)

type Consumer struct {
	js      jetstream.JetStream
	subject string
}

type AnswerRecorder interface {
	RecordNatsAnswer(ctx context.Context, answer string)
}

func NewConsumer(js jetstream.JetStream, subject string) *Consumer {
	return &Consumer{
		js:      js,
		subject: subject,
	}
}

func ackMsg(msg jetstream.Msg) {
	if err := msg.Ack(); err != nil {
		log.Printf("failed to ack message: %v", err)
	}
}

func nakMsg(msg jetstream.Msg) {
	if err := msg.Nak(); err != nil {
		log.Printf("failed to nak message: %v", err)
	}
}

func termMsg(msg jetstream.Msg) {
	if err := msg.Term(); err != nil {
		log.Printf("failed to term message: %v", err)
	}
}

func (c *Consumer) Consume(appCtx context.Context, handler func(context.Context, []byte, nats.Header) error,
	errHandler func(context.Context, []byte, nats.Header), groupName string, answerRecorder AnswerRecorder) error {

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
		if err == nil {
			ackMsg(msg)
			recordAnswer(msgCtx, answerRecorder, "ack")
		} else {
			if errors.Is(err, TermError) {
				errCtx, cancelErr := context.WithTimeout(appCtx, 10*time.Minute)
				defer cancelErr()
				errHandler(errCtx, binData, header)
				termMsg(msg)
				recordAnswer(msgCtx, answerRecorder, "term")
			} else {
				nakMsg(msg)
				recordAnswer(msgCtx, answerRecorder, "nak")
			}
		}
	})

	if err != nil {
		return fmt.Errorf("start consume: %w", err)
	}
	defer cc.Stop()

	<-appCtx.Done()
	return nil
}

func recordAnswer(ctx context.Context, recorder AnswerRecorder, answer string) {
	if recorder == nil {
		return
	}

	recorder.RecordNatsAnswer(ctx, answer)
}
