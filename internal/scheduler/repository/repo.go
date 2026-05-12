package repository

import (
	"context"

	"github.com/google/uuid"
)

type PostgresRepo interface {
	ClaimNextJobs(ctx context.Context, jobBatchSize int) ([]uuid.UUID, error)
	ResetHungMessage(ctx context.Context, scheduleJobTimeout int, procJobTimeout int) error
	SwitchToDisabledIfNeed(ctx context.Context) error
}
