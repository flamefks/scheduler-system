package nats

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	logger  *slog.Logger
}

type AnswerRecorder interface {
	RecordNatsAnswer(ctx context.Context, answerType string, status string)
}

func NewConsumer(js jetstream.JetStream, subject string, logger *slog.Logger) *Consumer {
	return &Consumer{
		js:      js,
		subject: subject,
		logger:  logger,
	}
}

func ackMsg(msg jetstream.Msg) error {
	if err := msg.Ack(); err != nil {
		slog.Error("failed_ack_message", slog.Any("error", err))
		return err
	}
	return nil
}

func nakMsg(msg jetstream.Msg) error {
	if err := msg.Nak(); err != nil {
		slog.Error("failed_nak_message", slog.Any("error", err))
		return err
	}
	return nil
}

func termMsg(msg jetstream.Msg) error {
	if err := msg.Term(); err != nil {
		slog.Error("failed_term_message", slog.Any("error", err))
		return err
	}
	return nil
}

func (c *Consumer) Consume(appCtx context.Context, handler func(context.Context, []byte, nats.Header, uint64) error,
	errHandler func(context.Context, []byte, nats.Header), groupName string, answerRecorder AnswerRecorder) error {

	initCtx, cancel := context.WithTimeout(appCtx, 5*time.Second)
	defer cancel()
	stream, err := c.js.Stream(initCtx, sharedData.NatsStreamName)
	if err != nil {
		slog.Error("worker_stream_error", slog.Any("error", err))
		return fmt.Errorf("worker: stream error: %v", err)
	}

	consumer, err := stream.Consumer(initCtx, groupName)
	if err != nil {
		return fmt.Errorf("worker: consumer error: %v", err)
	}

	info, err := consumer.Info(initCtx)
	if err != nil {
		return fmt.Errorf("worker: consumer info error: %w", err)
	}

	maxDeliver := info.Config.MaxDeliver

	cc, err := consumer.Consume(func(msg jetstream.Msg) {
		msgCtx, cancel := context.WithTimeout(appCtx, 2*time.Hour)
		defer cancel()

		binData := msg.Data()
		header := msg.Headers()
		deliveryAttempt := deliveryAttemptFromMetadata(msg)
		err := handler(msgCtx, binData, header, deliveryAttempt)
		if err == nil {
			recordAnswer(msgCtx, answerRecorder, "ack", ackMsg(msg))
		} else {
			if errors.Is(err, TermError) || (deliveryAttempt == uint64(maxDeliver)) {
				errCtx, cancelErr := context.WithTimeout(appCtx, 10*time.Minute)
				defer cancelErr()
				errHandler(errCtx, binData, header)
				recordAnswer(msgCtx, answerRecorder, "term", termMsg(msg))
			} else {
				recordAnswer(msgCtx, answerRecorder, "nak", nakMsg(msg))
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

func deliveryAttemptFromMetadata(msg jetstream.Msg) uint64 {
	metadata, err := msg.Metadata()
	if err != nil {
		slog.Error("failed_read_message_metadata", slog.Any("error", err))
		return 0
	}
	return metadata.NumDelivered
}

func recordAnswer(ctx context.Context, recorder AnswerRecorder, answerType string, err error) {
	if recorder == nil {
		return
	}

	status := "success"
	if err != nil {
		status = "error"
	}
	recorder.RecordNatsAnswer(ctx, answerType, status)
}
