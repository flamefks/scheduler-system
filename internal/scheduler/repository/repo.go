package repository

import (
	"context"

	"github.com/google/uuid"
)

type PostgresRepo interface {
	ClaimNextJobs(ctx context.Context, jobBatchSize int) ([]uuid.UUID, error)
	ResetHungMessage(ctx context.Context, scheduleJobTimeout int, procJobTimeout int) (int64, error)
	SwitchToDisabledIfNeed(ctx context.Context) (int64, error)
}
