package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	OperationCreateJob     = "create_job"
	OperationGetJob        = "get_job"
	OperationPatchJob      = "patch_job"
	OperationDeleteJob     = "delete_job"
	OperationActivateJob   = "activate_job"
	OperationDeactivateJob = "deactivate_job"
)

type ApiMetrics struct {
	httpTotal    metric.Int64Counter
	httpDuration metric.Float64Histogram

	dbOperationTotal metric.Int64Counter
}

func NewApiMetrics() (*ApiMetrics, error) {
	meter := otel.Meter("scheduler-system/api")

	httpTotal, err := meter.Int64Counter("http_api_responses_total")
	if err != nil {
		return nil, err
	}
	httpDuration, err := meter.Float64Histogram(
		"http_api_duration_seconds",
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	dbOperationTotal, err := meter.Int64Counter("api_db_operations_total")
	if err != nil {
		return nil, err
	}

	return &ApiMetrics{
		httpTotal:        httpTotal,
		httpDuration:     httpDuration,
		dbOperationTotal: dbOperationTotal,
	}, nil
}

func (m *ApiMetrics) RecordHTTP(ctx context.Context, handler string, statusCode int, duration time.Duration) {
	if m == nil { // test depend
		return
	}

	attrs := metric.WithAttributes(
		attribute.String("handler", handler),
		attribute.Int("status_code", statusCode),
	)

	m.httpTotal.Add(ctx, 1, attrs)
	m.httpDuration.Record(ctx, duration.Seconds(), attrs)
}

func (m *ApiMetrics) RecordDBOperation(ctx context.Context, operation string, status string) {
	if m == nil {
		return
	}

	m.dbOperationTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("operation", operation),
		attribute.String("status", status),
	))
}
