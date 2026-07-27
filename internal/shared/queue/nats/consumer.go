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

type Handler func(context.Context, []byte, nats.Header, uint64, time.Duration) error
type ErrorHandler func(context.Context, []byte, nats.Header) error

func NewConsumer(js jetstream.JetStream, subject string, logger *slog.Logger) *Consumer {
	if logger == nil {
		logger = slog.Default()
	}

	return &Consumer{
		js:      js,
		subject: subject,
		logger:  logger,
	}
}

func (c *Consumer) ackMsg(msg jetstream.Msg) error {
	if err := msg.Ack(); err != nil {
		c.logger.Error("failed_ack_message", slog.Any("error", err))
		return err
	}
	return nil
}

func (c *Consumer) nakMsg(msg jetstream.Msg) error {
	if err := msg.Nak(); err != nil {
		c.logger.Error("failed_nak_message", slog.Any("error", err))
		return err
	}
	return nil
}

func (c *Consumer) termMsg(msg jetstream.Msg) error {
	if err := msg.Term(); err != nil {
		c.logger.Error("failed_term_message", slog.Any("error", err))
		return err
	}
	return nil
}

func (c *Consumer) Consume(appCtx context.Context, handler Handler,
	errHandler ErrorHandler, groupName string, answerRecorder AnswerRecorder) error {

	initCtx, cancel := context.WithTimeout(appCtx, 5*time.Second)
	defer cancel()
	stream, err := c.js.Stream(initCtx, sharedData.NatsStreamName)
	if err != nil {
		c.logger.Error("worker_stream_error", slog.Any("error", err))
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
	ackWait := info.Config.AckWait
	maxProcessingTime := ackWait
	if maxProcessingTime <= 0 && len(info.Config.BackOff) > 0 {
		maxProcessingTime = info.Config.BackOff[0]
	}
	if maxProcessingTime <= 0 {
		maxProcessingTime = 2 * time.Minute
	}
	c.logger.Info(
		"consumer_config_loaded",
		slog.String("subject", c.subject),
		slog.String("group", groupName),
		slog.Duration("ack_wait", ackWait),
		slog.Duration("max_processing_time", maxProcessingTime),
		slog.Int("max_deliver", maxDeliver),
		slog.Int("max_ack_pending", info.Config.MaxAckPending),
		slog.Any("backoff", info.Config.BackOff),
	)

	cc, err := consumer.Consume(func(msg jetstream.Msg) {
		msgCtx, cancel := context.WithTimeout(appCtx, maxProcessingTime)
		defer cancel()

		binData := msg.Data()
		header := msg.Headers()
		deliveryAttempt := c.deliveryAttemptFromMetadata(msg)
		err := handler(msgCtx, binData, header, deliveryAttempt, maxProcessingTime)
		if err == nil {
			recordAnswer(msgCtx, answerRecorder, "ack", c.ackMsg(msg))
		} else {
			maxDeliverReached := maxDeliver > 0 && deliveryAttempt >= uint64(maxDeliver)
			if errors.Is(err, TermError) || maxDeliverReached {
				errCtx, cancelErr := context.WithTimeout(appCtx, 10*time.Minute)
				defer cancelErr()
				err := errHandler(errCtx, binData, header)
				if err == nil || maxDeliverReached {
					recordAnswer(msgCtx, answerRecorder, "term", c.termMsg(msg))
				} else {
					recordAnswer(msgCtx, answerRecorder, "nak", c.nakMsg(msg))
				}
			} else {
				recordAnswer(msgCtx, answerRecorder, "nak", c.nakMsg(msg))
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

func (c *Consumer) deliveryAttemptFromMetadata(msg jetstream.Msg) uint64 {
	metadata, err := msg.Metadata()
	if err != nil {
		c.logger.Error("failed_read_message_metadata", slog.Any("error", err))
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
