package shared

import (
	"context"
	"fmt"

	db "github.com/flamefks/scheduler-system/internal/postgres/queries"
	"github.com/flamefks/scheduler-system/internal/shared/data"
	"github.com/google/uuid"
)

type WorkerRepository struct {
	q *db.Queries
}

func NewWorkerRepository(q *db.Queries) *WorkerRepository {
	return &WorkerRepository{
		q: q,
	}
}

func (repo *WorkerRepository) GetConfig(ctx context.Context, kind string, jobId uuid.UUID) (*data.IOConfig, error) {
	jKind, err := getJobKindEnum(kind)
	if err != nil {
		return nil, err
	}

	ioConfig, err := repo.q.GetConfig(ctx, db.GetConfigParams{
		JobID: jobId,
		Kind:  jKind,
	})
	if err != nil {
		return nil, fmt.Errorf("failed get JobIoConfig: %w", err)
	}
	return &data.IOConfig{
		TargetUrl:  ioConfig.TargetUrl,
		Method:     ioConfig.Method,
		Payload:    ioConfig.Payload,
		Headers:    ioConfig.Headers,
		JsonSchema: ioConfig.JsonSchema,
	}, nil
}

func (repo *WorkerRepository) SetJobStatus(ctx context.Context, status string, jobId uuid.UUID) error {
	jStatus, err := getJobStatusEnum(status)
	if err != nil {
		return err
	}

	return repo.q.SetJobStatus(ctx, db.SetJobStatusParams{
		Status: jStatus,
		JobID:  jobId,
	})
}

// helper
func getJobKindEnum(rowKind string) (db.JobIoKind, error) {
	jKind := db.JobIoKind(rowKind)
	switch jKind {
	case db.JobIoKindFetcher, db.JobIoKindDeliver:
		return jKind, nil
	default:
		return "", fmt.Errorf("invalid JobIoKind: %s", rowKind)
	}
}

func getJobStatusEnum(jStatus string) (db.ScheduleStatus, error) {
	jobStatusEnum := db.ScheduleStatus(jStatus)
	switch jobStatusEnum {
	case db.ScheduleStatusIdle, db.ScheduleStatusScheduled, db.ScheduleStatusFetching, db.ScheduleStatusDelivering,
		db.ScheduleStatusError, db.ScheduleStatusDisabled:
		return jobStatusEnum, nil
	default:
		return "", fmt.Errorf("invalid JobStatus: %s", jStatus)
	}
}
