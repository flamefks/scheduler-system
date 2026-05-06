package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/flamefks/scheduler-system/internal/api/domain"
	db "github.com/flamefks/scheduler-system/internal/postgres/queries"
	"github.com/flamefks/scheduler-system/internal/shared/data"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewRepository(pool *pgxpool.Pool, q *db.Queries) *Repository {
	return &Repository{
		pool: pool,
		q:    q,
	}
}

// =========================
// CREATE
// =========================

func (repo *Repository) CreateJob(ctx context.Context, job *data.Job) (uuid.UUID, error) {
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := repo.q.WithTx(tx)

	jobID := uuid.New()

	// jobs
	if _, err := qtx.CreateJob(ctx, db.CreateJobParams{
		ID:   jobID,
		Name: job.Name,
	}); err != nil {
		return uuid.Nil, fmt.Errorf("create job: %w", err)
	}

	// schedule
	if err := qtx.CreateJobSchedule(ctx, db.CreateJobScheduleParams{
		JobID:             jobID,
		NextRunAt:         job.Schedule.NextRunAt,
		RepeatIntervalSec: job.Schedule.RepeatIntervalSec,
		TargetRuns:        job.Schedule.TargetRuns,
	}); err != nil {
		return uuid.Nil, fmt.Errorf("create schedule: %w", err)
	}

	// fetcher config
	if err := qtx.CreateJobIOConfig(ctx, db.CreateJobIOConfigParams{
		JobID:     jobID,
		Kind:      db.JobIoKindFetcher,
		Payload:   job.FetcherConfig.Payload,
		Headers:   job.FetcherConfig.Headers,
		TargetUrl: job.FetcherConfig.TargetUrl,
		Method:    job.FetcherConfig.Method,
	}); err != nil {
		return uuid.Nil, fmt.Errorf("create fetcher config: %w", err)
	}

	// deliver config
	if err := qtx.CreateJobIOConfig(ctx, db.CreateJobIOConfigParams{
		JobID:     jobID,
		Kind:      db.JobIoKindDeliver,
		Payload:   job.DeliverConfig.Payload,
		Headers:   job.DeliverConfig.Headers,
		TargetUrl: job.DeliverConfig.TargetUrl,
		Method:    job.DeliverConfig.Method,
	}); err != nil {
		return uuid.Nil, fmt.Errorf("create deliver config: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("commit: %w", err)
	}

	return jobID, nil
}

// =========================
// GET
// =========================

func (repo *Repository) GetJobByID(ctx context.Context, id uuid.UUID) (*data.Job, error) {
	jobDb, err := repo.q.GetJob(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}

	schedule, err := repo.q.GetJobSchedule(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get schedule: %w", err)
	}

	configs, err := repo.q.ListJobIOConfigs(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get configs: %w", err)
	}

	var fetcher, deliver data.IOConfig

	for _, c := range configs {
		switch c.Kind {
		case db.JobIoKindFetcher:
			fetcher = data.IOConfig{
				Payload:   c.Payload,
				Headers:   c.Headers,
				TargetUrl: c.TargetUrl,
				Method:    c.Method,
			}
		case db.JobIoKindDeliver:
			deliver = data.IOConfig{
				Payload:   c.Payload,
				Headers:   c.Headers,
				TargetUrl: c.TargetUrl,
				Method:    c.Method,
			}
		}
	}

	return &data.Job{
		ID:   jobDb.ID,
		Name: jobDb.Name,

		Schedule: data.Schedule{
			Status:            string(schedule.Status),
			RepeatIntervalSec: schedule.RepeatIntervalSec,
			TargetRuns:        schedule.TargetRuns,
			ScheduledRuns:     schedule.ScheduledRuns,
			NextRunAt:         schedule.NextRunAt,
			LastRunAt:         schedule.LastRunAt,
		},

		FetcherConfig: fetcher,
		DeliverConfig: deliver,

		CreatedAt: jobDb.CreatedAt,
		UpdatedAt: jobDb.UpdatedAt,
	}, nil
}

// =========================
// PATCH
// =========================

func (repo *Repository) PatchJob(ctx context.Context, patch *domain.PatchJobModel, id uuid.UUID) error {
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := repo.q.WithTx(tx)

	// jobs
	if patch.Name != nil {
		if _, err := qtx.UpdateJobName(ctx, db.UpdateJobNameParams{
			ID:   id,
			Name: *patch.Name,
		}); err != nil {
			return fmt.Errorf("update job: %w", err)
		}
	}

	// schedule
	if patch.Schedule != nil {
		var nextRunAt time.Time
		var setRunAt bool

		if patch.Schedule.NextRunAt != nil {
			setRunAt = true
			nextRunAt = *patch.Schedule.NextRunAt
		}

		var setScheduleStatus bool = false
		var statusAsEnum db.ScheduleStatus
		if patch.Schedule.Status != nil {
			setScheduleStatus = true
			statusAsEnum, err = getJobStatusEnum(*patch.Schedule.Status)
			if err != nil {
				return fmt.Errorf("patch schedule error: %w", err)
			}
		}

		if _, err := qtx.PatchJobSchedule(ctx, db.PatchJobScheduleParams{
			NextRunAt:         nextRunAt,
			SetNextRunAt:      setRunAt,
			RepeatIntervalSec: patch.Schedule.RepeatIntervalSec,
			TargetRuns:        patch.Schedule.TargetRuns,
			Status: db.NullScheduleStatus{
				ScheduleStatus: statusAsEnum,
				Valid:          setScheduleStatus,
			},
			JobID: id,
		}); err != nil {
			return fmt.Errorf("patch schedule: %w", err)
		}
	}

	// fetcher config
	if patch.FetcherConfig != nil {
		var SetPayload bool = false
		var SetHeaders bool = false

		var payload []byte
		var Headers []byte

		if patch.FetcherConfig.Headers != nil {
			SetHeaders = true
			Headers = *patch.FetcherConfig.Headers
		}
		if patch.FetcherConfig.Payload != nil {
			SetPayload = true
			payload = *patch.FetcherConfig.Payload
		}

		if _, err := qtx.PatchJobIOConfig(ctx, db.PatchJobIOConfigParams{
			JobID:      id,
			Kind:       db.JobIoKindFetcher,
			SetPayload: SetPayload,
			Payload:    payload,
			SetHeaders: SetHeaders,
			Headers:    Headers,
			TargetUrl:  patch.FetcherConfig.TargetUrl,
			Method:     patch.FetcherConfig.Method,
		}); err != nil {
			return fmt.Errorf("patch fetcher config: %w", err)
		}
	}

	// deliver config
	if patch.DeliverConfig != nil {
		var SetPayload bool = false
		var SetHeaders bool = false

		var payload []byte
		var Headers []byte

		if patch.DeliverConfig.Headers != nil {
			SetHeaders = true
			Headers = *patch.DeliverConfig.Headers
		}
		if patch.DeliverConfig.Payload != nil {
			SetPayload = true
			payload = *patch.DeliverConfig.Payload
		}
		if _, err := qtx.PatchJobIOConfig(ctx, db.PatchJobIOConfigParams{
			JobID:      id,
			Kind:       db.JobIoKindDeliver,
			SetPayload: SetPayload,
			Payload:    payload,
			SetHeaders: SetHeaders,
			Headers:    Headers,
			TargetUrl:  patch.DeliverConfig.TargetUrl,
			Method:     patch.DeliverConfig.Method,
		}); err != nil {
			return fmt.Errorf("patch deliver config: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

// =========================
// DELETE
// =========================

func (repo *Repository) DeleteJob(ctx context.Context, id uuid.UUID) error {
	return repo.q.DeleteJob(ctx, id)
}

// =========================
// UpdateScheduleStatus
// =========================
func (repo *Repository) UpdateScheduleStatus(ctx context.Context, id uuid.UUID, status string) error {
	schedulerStatus, err := getJobStatusEnum(status)
	if err != nil {
		return err
	}
	err = repo.q.UpdateJobScheduleStatus(ctx, db.UpdateJobScheduleStatusParams{
		Status: schedulerStatus,
		JobID:  id,
	})
	if err != nil {
		return fmt.Errorf("Failed update job status: %w", err)
	}
	return nil
}

// =========================
// HELPERS
// =========================
func getJobStatusEnum(rowStatus string) (db.ScheduleStatus, error) {
	scheduleStatus := db.ScheduleStatus(rowStatus)
	switch scheduleStatus {
	case db.ScheduleStatusIdle, db.ScheduleStatusRunning, db.ScheduleStatusError, db.ScheduleStatusDisabled:
		return scheduleStatus, nil
	default:
		return "", fmt.Errorf("invalid scheduleStatus: %s", rowStatus)
	}
}
