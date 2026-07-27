package repository

import (
	"context"

	"github.com/flamefks/scheduler-system/internal/shared/data"
	"github.com/google/uuid"
)

type PostgresRepo interface {
	GetConfig(ctx context.Context, kind string, jobId uuid.UUID) (*data.IOConfig, error)
	SetJobRunStatus(ctx context.Context, status string, jobId uuid.UUID, runId uuid.UUID) error
}
