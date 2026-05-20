package metrics

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type DeliveryMetrics struct {
	natsAnswersTotal metric.Int64Counter

	httpRequestTotal metric.Int64Counter

	deliveryJobsTotal metric.Int64Counter

	errorHandlerTotal     metric.Int64Counter
	errorHandlerJobsTotal metric.Int64Counter
}

func NewDeliveryMetrics() (*DeliveryMetrics, error) {
	meter := otel.Meter("scheduler-system/delivery")

	natsAnswersTotal, err := meter.Int64Counter("nats_answers_total")
	if err != nil {
		return nil, err
	}

	httpRequestTotal, err := meter.Int64Counter("delivery_http_requests_total")
	if err != nil {
		return nil, err
	}

	deliveryJobsTotal, err := meter.Int64Counter("delivery_jobs_total")
	if err != nil {
		return nil, err
	}

	errorHandlerTotal, err := meter.Int64Counter("delivery_error_handler_total")
	if err != nil {
		return nil, err
	}

	errorHandlerJobsTotal, err := meter.Int64Counter("delivery_error_handler_jobs_total")
	if err != nil {
		return nil, err
	}

	return &DeliveryMetrics{
		natsAnswersTotal: natsAnswersTotal,

		httpRequestTotal: httpRequestTotal,

		deliveryJobsTotal: deliveryJobsTotal,

		errorHandlerTotal:     errorHandlerTotal,
		errorHandlerJobsTotal: errorHandlerJobsTotal,
	}, nil
}

func (m *DeliveryMetrics) RecordNatsAnswer(ctx context.Context, answer string) {
	if m == nil {
		return
	}

	m.natsAnswersTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("answer", answer),
	))
}

func (m *DeliveryMetrics) RecordHTTPRequest(ctx context.Context, status string, statusCode int) {
	if m == nil {
		return
	}

	m.httpRequestTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("status", status),
		attribute.Int("status_code", statusCode),
	))
}

func (m *DeliveryMetrics) RecordDeliveryJobs(ctx context.Context, count int64) {
	if m == nil || count <= 0 {
		return
	}

	m.deliveryJobsTotal.Add(ctx, count)
}

func (m *DeliveryMetrics) RecordErrorHandler(ctx context.Context, status string) {
	if m == nil {
		return
	}

	m.errorHandlerTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("status", status),
	))
}

func (m *DeliveryMetrics) RecordErrorHandlerJobs(ctx context.Context, count int64) {
	if m == nil || count <= 0 {
		return
	}

	m.errorHandlerJobsTotal.Add(ctx, count)
}
