package repository

import (
	"context"

	"github.com/flamefks/scheduler-system/internal/shared/data"
)

type PostgresRepo interface {
	ClaimNextJobs(ctx context.Context, jobBatchSize int) ([]data.JobRun, error)
	ResetHungMessage(ctx context.Context, scheduleJobTimeout int, procJobTimeout int) (int64, error)
	SwitchToDisabledIfNeed(ctx context.Context) (int64, error)
}
