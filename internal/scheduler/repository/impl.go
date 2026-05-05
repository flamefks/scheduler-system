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

func (repo *SchedulerRepository) ClaimNextJob(ctx context.Context) (uuid.UUID, error) {
	return repo.q.ClaimNextJob(ctx)
}

func (repo *SchedulerRepository) ResetHungMessage(ctx context.Context, JobDeathTimeout int) error {
	return repo.q.ResetHungMessage(ctx, int64(JobDeathTimeout))
}

func (repo *SchedulerRepository) SwitchToDisabledIfNeed(ctx context.Context) error {
	return repo.q.SwitchToDisabledIfNeed(ctx)
}
