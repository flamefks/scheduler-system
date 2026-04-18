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
