package metrics

import (
	"context"

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
	httpTotal         metric.Int64Counter
	httpCreateJob     metric.Int64Counter
	httpGetJob        metric.Int64Counter
	httpPatchJob      metric.Int64Counter
	httpDeleteJob     metric.Int64Counter
	httpActivateJob   metric.Int64Counter
	httpDeactivateJob metric.Int64Counter
}

func NewApiMetrics() (*ApiMetrics, error) {
	meter := otel.Meter("scheduler-system/api")

	httpTotal, err := meter.Int64Counter("http_responses_total")
	if err != nil {
		return nil, err
	}

	httpCreateJob, err := meter.Int64Counter("http_create_job_total")
	if err != nil {
		return nil, err
	}

	httpGetJob, err := meter.Int64Counter("http_get_job_total")
	if err != nil {
		return nil, err
	}

	httpPatchJob, err := meter.Int64Counter("http_patch_job_total")
	if err != nil {
		return nil, err
	}

	httpDeleteJob, err := meter.Int64Counter("http_delete_job_total")
	if err != nil {
		return nil, err
	}

	httpActivateJob, err := meter.Int64Counter("http_activate_job_total")
	if err != nil {
		return nil, err
	}

	httpDeactivateJob, err := meter.Int64Counter("http_deactivate_job_total")
	if err != nil {
		return nil, err
	}

	return &ApiMetrics{
		httpTotal:         httpTotal,
		httpCreateJob:     httpCreateJob,
		httpGetJob:        httpGetJob,
		httpPatchJob:      httpPatchJob,
		httpDeleteJob:     httpDeleteJob,
		httpActivateJob:   httpActivateJob,
		httpDeactivateJob: httpDeactivateJob,
	}, nil
}

func (m *ApiMetrics) Record(ctx context.Context, operation string, err error) {
	if m == nil {
		return
	}

	result := "success"
	if err != nil {
		result = "error"
	}

	m.httpTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("operation", operation),
		attribute.String("result", result),
	))

	attrs := metric.WithAttributes(attribute.String("result", result))
	switch operation {
	case OperationCreateJob:
		m.httpCreateJob.Add(ctx, 1, attrs)
	case OperationGetJob:
		m.httpGetJob.Add(ctx, 1, attrs)
	case OperationPatchJob:
		m.httpPatchJob.Add(ctx, 1, attrs)
	case OperationDeleteJob:
		m.httpDeleteJob.Add(ctx, 1, attrs)
	case OperationActivateJob:
		m.httpActivateJob.Add(ctx, 1, attrs)
	case OperationDeactivateJob:
		m.httpDeactivateJob.Add(ctx, 1, attrs)
	}
}
