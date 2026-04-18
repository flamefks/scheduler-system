package repository

import (
	"context"

	"github.com/flamefks/scheduler-system/internal/shared"
	"github.com/google/uuid"
)

type PostgresRepo interface {
	GetConfig(ctx context.Context, kind string, jobId uuid.UUID) (*shared.IOConfig, error)
}
