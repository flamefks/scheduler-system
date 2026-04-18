package repository

import (
	"context"

	"github.com/flamefks/scheduler-system/internal/api/domain"
	"github.com/flamefks/scheduler-system/internal/shared"
	"github.com/google/uuid"
)

type PostgresRepo interface {
	CreateJob(ctx context.Context, job *shared.Job) (uuid.UUID, error)
	DeleteJob(ctx context.Context, id uuid.UUID) error
	GetJobByID(ctx context.Context, id uuid.UUID) (*shared.Job, error)
	PatchJob(ctx context.Context, patch *domain.PatchJobModel, id uuid.UUID) error
	UpdateScheduleStatus(ctx context.Context, id uuid.UUID, status string) error
}
