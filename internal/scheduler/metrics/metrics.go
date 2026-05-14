package metrics

import (
	"go.opentelemetry.io/otel/metric"
)

type SchedulerMetrics struct {
	httpTotal    metric.Int64Counter
	httpDuration metric.Float64Histogram
}
