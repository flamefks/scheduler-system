package repository

import (
	"context"

	db "github.com/flamefks/scheduler-system/internal/postgres/queries"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SchedulerRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewSchedulerRepository(pool *pgxpool.Pool, q *db.Queries) *SchedulerRepository {
	return &SchedulerRepository{
		pool: pool,
		q:    q,
	}
}

func (repo *SchedulerRepository) ClaimNextJobs(ctx context.Context, jobBatchSize int) ([]uuid.UUID, error) {
	return repo.q.ClaimNextJobs(ctx, int32(jobBatchSize))
}

func (repo *SchedulerRepository) ResetHungMessage(ctx context.Context, scheduleJobTimeout int, procJobTimeout int) error {
	return repo.q.ResetHungMessage(ctx, db.ResetHungMessageParams{
		ScheduleTimeoutSeconds: int64(scheduleJobTimeout),
		ProcTimeoutSeconds:     int64(procJobTimeout),
	})
}

func (repo *SchedulerRepository) SwitchToDisabledIfNeed(ctx context.Context) error {
	return repo.q.SwitchToDisabledIfNeed(ctx)
}
