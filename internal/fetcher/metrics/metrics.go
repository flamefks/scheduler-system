package metrics

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type FetcherMetrics struct {
	natsAnswersTotal metric.Int64Counter

	httpRequestTotal metric.Int64Counter

	natsPublishTotal     metric.Int64Counter
	natsPublishJobsTotal metric.Int64Counter

	errorHandlerTotal     metric.Int64Counter
	errorHandlerJobsTotal metric.Int64Counter
}

func NewFetcherMetrics() (*FetcherMetrics, error) {
	meter := otel.Meter("scheduler-system/fetcher")

	natsAnswersTotal, err := meter.Int64Counter("nats_answers_total")
	if err != nil {
		return nil, err
	}

	httpRequestTotal, err := meter.Int64Counter("fetcher_http_requests_total")
	if err != nil {
		return nil, err
	}

	natsPublishTotal, err := meter.Int64Counter("fetcher_nats_publish_total")
	if err != nil {
		return nil, err
	}

	natsPublishJobsTotal, err := meter.Int64Counter("fetcher_nats_publish_jobs_total")
	if err != nil {
		return nil, err
	}

	errorHandlerTotal, err := meter.Int64Counter("fetcher_error_handler_total")
	if err != nil {
		return nil, err
	}

	errorHandlerJobsTotal, err := meter.Int64Counter("fetcher_error_handler_jobs_total")
	if err != nil {
		return nil, err
	}

	return &FetcherMetrics{
		natsAnswersTotal: natsAnswersTotal,

		httpRequestTotal: httpRequestTotal,

		natsPublishTotal:     natsPublishTotal,
		natsPublishJobsTotal: natsPublishJobsTotal,

		errorHandlerTotal:     errorHandlerTotal,
		errorHandlerJobsTotal: errorHandlerJobsTotal,
	}, nil
}

func (m *FetcherMetrics) RecordNatsAnswer(ctx context.Context, answer string) {
	if m == nil {
		return
	}

	m.natsAnswersTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("answer", answer),
	))
}

func (m *FetcherMetrics) RecordHTTPRequest(ctx context.Context, status string, statusCode int) {
	if m == nil {
		return
	}

	m.httpRequestTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("status", status),
		attribute.Int("status_code", statusCode),
	))
}

func (m *FetcherMetrics) RecordNatsPublish(ctx context.Context, status string) {
	if m == nil {
		return
	}

	m.natsPublishTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("status", status),
	))
}

func (m *FetcherMetrics) RecordNatsPublishJobs(ctx context.Context, count int64) {
	if m == nil || count <= 0 {
		return
	}

	m.natsPublishJobsTotal.Add(ctx, count)
}

func (m *FetcherMetrics) RecordErrorHandler(ctx context.Context, status string) {
	if m == nil {
		return
	}

	m.errorHandlerTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("status", status),
	))
}

func (m *FetcherMetrics) RecordErrorHandlerJobs(ctx context.Context, count int64) {
	if m == nil || count <= 0 {
		return
	}

	m.errorHandlerJobsTotal.Add(ctx, count)
}
