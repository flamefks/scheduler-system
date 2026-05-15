package metrics

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type SchedulerMetrics struct {
	claimedTotal     metric.Int64Counter
	claimedJobsTotal metric.Int64Counter

	resetHungTotal     metric.Int64Counter
	resetHungJobsTotal metric.Int64Counter

	disabledTotal     metric.Int64Counter
	disabledJobsTotal metric.Int64Counter

	natsPublishTotal metric.Int64Counter
}

func NewSchedulerMetrics() (*SchedulerMetrics, error) {
	meter := otel.Meter("scheduler-system/scheduler")

	claimedTotal, err := meter.Int64Counter("scheduler_claimed_total")
	if err != nil {
		return nil, err
	}

	claimedJobsTotal, err := meter.Int64Counter("scheduler_claimed_jobs_total")
	if err != nil {
		return nil, err
	}

	resetHungTotal, err := meter.Int64Counter("scheduler_hung_reset_total")
	if err != nil {
		return nil, err
	}

	resetHungJobsTotal, err := meter.Int64Counter("scheduler_hung_jobs_reset_total")
	if err != nil {
		return nil, err
	}

	disabledTotal, err := meter.Int64Counter("scheduler_disabled_total")
	if err != nil {
		return nil, err
	}

	disabledJobsTotal, err := meter.Int64Counter("scheduler_jobs_disabled_total")
	if err != nil {
		return nil, err
	}

	natsPublishTotal, err := meter.Int64Counter("scheduler_nats_publish_total")
	if err != nil {
		return nil, err
	}

	return &SchedulerMetrics{
		claimedTotal:     claimedTotal,
		claimedJobsTotal: claimedJobsTotal,

		resetHungTotal:     resetHungTotal,
		resetHungJobsTotal: resetHungJobsTotal,

		disabledTotal:     disabledTotal,
		disabledJobsTotal: disabledJobsTotal,

		natsPublishTotal: natsPublishTotal,
	}, nil
}

func (m *SchedulerMetrics) RecordClaimed(ctx context.Context, status string) {
	if m == nil {
		return
	}

	m.claimedTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("status", status),
	))
}

func (m *SchedulerMetrics) RecordClaimedJobs(ctx context.Context, claimedCount int) {
	if m == nil || claimedCount <= 0 {
		return
	}

	m.claimedJobsTotal.Add(ctx, int64(claimedCount))
}

func (m *SchedulerMetrics) RecordResetHung(ctx context.Context, status string) {
	if m == nil {
		return
	}

	m.resetHungTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("status", status),
	))
}

func (m *SchedulerMetrics) RecordResetHungJobs(ctx context.Context, count int64) {
	if m == nil || count <= 0 {
		return
	}

	m.resetHungJobsTotal.Add(ctx, count)
}

func (m *SchedulerMetrics) RecordDisabled(ctx context.Context, status string) {
	if m == nil {
		return
	}

	m.disabledTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("status", status),
	))
}

func (m *SchedulerMetrics) RecordDisabledJobs(ctx context.Context, count int64) {
	if m == nil || count <= 0 {
		return
	}

	m.disabledJobsTotal.Add(ctx, count)
}

func (m *SchedulerMetrics) RecordNatsPublish(ctx context.Context, status string) {
	if m == nil {
		return
	}

	m.natsPublishTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("status", status),
	))
}
