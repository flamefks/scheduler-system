package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/flamefks/scheduler-system/internal/api/apperrors"
	"github.com/flamefks/scheduler-system/internal/api/domain"
	db "github.com/flamefks/scheduler-system/internal/postgres/queries"
	"github.com/flamefks/scheduler-system/internal/shared/data"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
		return uuid.Nil, fmt.Errorf("begin_transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := repo.q.WithTx(tx)

	jobID := uuid.New()

	// jobs
	if _, err := qtx.CreateJob(ctx, db.CreateJobParams{
		ID:   jobID,
		Name: job.Name,
	}); err != nil {
		return uuid.Nil, fmt.Errorf("create_job_table: %w", err)
	}

	// schedule
	if err := qtx.CreateJobSchedule(ctx, db.CreateJobScheduleParams{
		JobID:             jobID,
		NextRunAt:         job.Schedule.NextRunAt,
		RepeatIntervalSec: job.Schedule.RepeatIntervalSec,
		TargetRuns:        job.Schedule.TargetRuns,
	}); err != nil {
		return uuid.Nil, fmt.Errorf("create_schedule_table: %w", err)
	}

	// fetcher config
	if err := qtx.CreateJobIOConfig(ctx, db.CreateJobIOConfigParams{
		JobID:      jobID,
		Kind:       db.JobIoKindFetcher,
		Payload:    job.FetcherConfig.Payload,
		Headers:    job.FetcherConfig.Headers,
		TargetUrl:  job.FetcherConfig.TargetUrl,
		Method:     job.FetcherConfig.Method,
		JsonSchema: job.FetcherConfig.JsonSchema,
	}); err != nil {
		return uuid.Nil, fmt.Errorf("create_fetcher_config_table: %w", err)
	}

	// deliver config
	if err := qtx.CreateJobIOConfig(ctx, db.CreateJobIOConfigParams{
		JobID:      jobID,
		Kind:       db.JobIoKindDeliver,
		Payload:    job.DeliverConfig.Payload,
		Headers:    job.DeliverConfig.Headers,
		TargetUrl:  job.DeliverConfig.TargetUrl,
		Method:     job.DeliverConfig.Method,
		JsonSchema: job.DeliverConfig.JsonSchema,
	}); err != nil {
		return uuid.Nil, fmt.Errorf("create_deliver_config_table: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("make_commit: %w", err)
	}

	return jobID, nil
}

// =========================
// GET
// =========================

func (repo *Repository) GetJobByID(ctx context.Context, id uuid.UUID) (*data.Job, error) {
	jobDb, err := repo.q.GetJob(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get_job: %w", apperrors.ErrNotFound)
		} else {
			return nil, err
		}
	}

	schedule, err := repo.q.GetJobSchedule(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get_schedule: %w", apperrors.ErrNotFound)
		} else {
			return nil, err
		}
	}

	configs, err := repo.q.ListJobIOConfigs(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get_configs: %w", apperrors.ErrNotFound)
		} else {
			return nil, err
		}
	}

	var fetcher, deliver data.IOConfig

	for _, c := range configs {
		switch c.Kind {
		case db.JobIoKindFetcher:
			fetcher = data.IOConfig{
				Payload:    c.Payload,
				Headers:    c.Headers,
				JsonSchema: c.JsonSchema,
				TargetUrl:  c.TargetUrl,
				Method:     c.Method,
			}
		case db.JobIoKindDeliver:
			deliver = data.IOConfig{
				Payload:    c.Payload,
				Headers:    c.Headers,
				JsonSchema: c.JsonSchema,
				TargetUrl:  c.TargetUrl,
				Method:     c.Method,
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
			DoneRuns:          schedule.DoneRuns,
			NextRunAt:         schedule.NextRunAt,
			LastScheduledAt:   schedule.LastScheduledAt,
			LastRunTakenAt:    schedule.LastRunTakenAt,
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
		return fmt.Errorf("begin_transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := repo.q.WithTx(tx)

	// jobs
	if patch.Name != nil {
		if _, err := qtx.UpdateJobName(ctx, db.UpdateJobNameParams{
			ID:   id,
			Name: *patch.Name,
		}); err != nil {
			return fmt.Errorf("patch_job_table: %w", err)
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

		if _, err := qtx.PatchJobSchedule(ctx, db.PatchJobScheduleParams{
			NextRunAt:         nextRunAt,
			SetNextRunAt:      setRunAt,
			RepeatIntervalSec: patch.Schedule.RepeatIntervalSec,
			TargetRuns:        patch.Schedule.TargetRuns,
			JobID:             id,
		}); err != nil {
			return fmt.Errorf("patch_schedule_table: %w", err)
		}
	}

	// fetcher config
	if patch.FetcherConfig != nil {
		if _, err := qtx.PatchJobIOConfig(ctx, db.PatchJobIOConfigParams{
			JobID:         id,
			Kind:          db.JobIoKindFetcher,
			SetPayload:    patch.FetcherConfig.Payload.Set,
			Payload:       patch.FetcherConfig.Payload.Value,
			SetHeaders:    patch.FetcherConfig.Headers.Set,
			Headers:       patch.FetcherConfig.Headers.Value,
			SetJsonSchema: patch.FetcherConfig.JsonSchema.Set,
			JsonSchema:    patch.FetcherConfig.JsonSchema.Value,
			TargetUrl:     patch.FetcherConfig.TargetUrl,
			Method:        patch.FetcherConfig.Method,
		}); err != nil {
			return fmt.Errorf("patch_fetcher_config_table: %w", err)
		}
	}

	// deliver config
	if patch.DeliverConfig != nil {
		if _, err := qtx.PatchJobIOConfig(ctx, db.PatchJobIOConfigParams{
			JobID:         id,
			Kind:          db.JobIoKindDeliver,
			SetPayload:    patch.DeliverConfig.Payload.Set,
			Payload:       patch.DeliverConfig.Payload.Value,
			SetHeaders:    patch.DeliverConfig.Headers.Set,
			Headers:       patch.DeliverConfig.Headers.Value,
			SetJsonSchema: patch.DeliverConfig.JsonSchema.Set,
			JsonSchema:    patch.DeliverConfig.JsonSchema.Value,
			TargetUrl:     patch.DeliverConfig.TargetUrl,
			Method:        patch.DeliverConfig.Method,
		}); err != nil {
			return fmt.Errorf("patch_deliver_config_table: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("make_commit: %w", err)
	}

	return nil
}

// =========================
// DELETE
// =========================

func (repo *Repository) DeleteJob(ctx context.Context, id uuid.UUID) error {
	_, err := repo.q.DeleteJob(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("job by id <%v> not found: %w", id, apperrors.ErrNotFound)
	} else {
		return err
	}
}

// =========================
// ACTIVATE
// =========================

func (repo *Repository) ActivateJob(ctx context.Context, id uuid.UUID) error {
	if _, err := repo.q.ActivateJob(ctx, id); err != nil {
		return repo.mapScheduleCommandError(ctx, id, "activate_job", err)
	}
	return nil
}

// =========================
// DEACTIVATE
// =========================

func (repo *Repository) DeactivateJob(ctx context.Context, id uuid.UUID) error {
	if _, err := repo.q.DeactivateJob(ctx, id); err != nil {
		return repo.mapScheduleCommandError(ctx, id, "deactivate_job", err)
	}
	return nil
}

// =========================
// HELPERS
// =========================
func (repo *Repository) mapScheduleCommandError(ctx context.Context, id uuid.UUID, operation string, err error) error {
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s: %w", operation, err)
	}

	if _, getErr := repo.q.GetJobSchedule(ctx, id); getErr != nil {
		if errors.Is(getErr, pgx.ErrNoRows) {
			return fmt.Errorf("%s: %w", operation, apperrors.ErrNotFound)
		}
		return fmt.Errorf("%s check_schedule: %w", operation, getErr)
	}

	return fmt.Errorf("%s: %w", operation, apperrors.ErrStatusConflict)
}
