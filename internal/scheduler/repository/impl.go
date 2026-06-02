package repository

import (
	"context"

	db "github.com/flamefks/scheduler-system/internal/postgres/queries"
	"github.com/flamefks/scheduler-system/internal/shared/data"
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

func (repo *SchedulerRepository) ClaimNextJobs(ctx context.Context, jobBatchSize int) ([]data.JobRun, error) {
	rows, err := repo.q.ClaimNextJobs(ctx, int32(jobBatchSize))
	if err != nil {
		return nil, err
	}

	jobRuns := make([]data.JobRun, 0, len(rows))
	for _, row := range rows {
		jobRuns = append(jobRuns, data.JobRun{
			JobID: row.JobID,
			RunID: row.RunID,
		})
	}
	return jobRuns, nil
}

func (repo *SchedulerRepository) ResetHungMessage(ctx context.Context, scheduleJobTimeout int, procJobTimeout int) (int64, error) {
	rAffected, err := repo.q.ResetHungMessage(ctx, db.ResetHungMessageParams{
		ScheduleTimeoutSeconds: int64(scheduleJobTimeout),
		ProcTimeoutSeconds:     int64(procJobTimeout),
	})
	return rAffected, err
}

func (repo *SchedulerRepository) SwitchToDisabledIfNeed(ctx context.Context) (int64, error) {
	rAffected, err := repo.q.SwitchToDisabledIfNeed(ctx)
	return rAffected, err
}
