package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/flamefks/scheduler-system/internal/api/domain"
	"github.com/flamefks/scheduler-system/internal/shared/data"
	"github.com/google/uuid"

	repo "github.com/flamefks/scheduler-system/internal/api/repository"
)

type ApiService struct {
	Logger *slog.Logger
	repo   repo.PostgresRepo
}

func NewApiService(Logger *slog.Logger, r repo.PostgresRepo) *ApiService {
	return &ApiService{
		Logger: Logger,
		repo:   r,
	}
}

func (service *ApiService) CreateJob(ctx context.Context, job *data.Job) (uuid.UUID, error) {
	jobId, err := service.repo.CreateJob(ctx, job)
	if err != nil {
		service.Logger.Error(
			"failed_create_job",
			slog.Any("job_data", job),
			slog.Any("error", err),
		)
		return uuid.Nil, fmt.Errorf("Error on creating job: %w", err)
	}
	service.Logger.Info(
		"success_create_job",
		slog.Any("job_id", job.ID),
		slog.Any("job_name", job.Name),
	)
	return jobId, nil
}

func (service *ApiService) DeleteJob(ctx context.Context, jobId uuid.UUID) error {
	err := service.repo.DeleteJob(ctx, jobId)
	if err != nil {
		service.Logger.Error(
			"failed_delete_job",
			slog.Any("job_id", jobId),
			slog.Any("error", err),
		)
		return fmt.Errorf("Error on removing job: %w", err)
	}
	service.Logger.Info(
		"success_delete_job",
		slog.Any("job_id", jobId),
	)
	return nil
}

func (service *ApiService) GetJobByID(ctx context.Context, jobId uuid.UUID) (*data.Job, error) {
	j, err := service.repo.GetJobByID(ctx, jobId)
	if err != nil {
		service.Logger.Error(
			"failed_get_job",
			slog.Any("job_id", jobId),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("error getting job by id %s", jobId)
	}
	service.Logger.Info(
		"success_get_job",
		slog.Any("job_id", jobId),
	)
	return j, nil
}

func (service *ApiService) PatchJob(ctx context.Context, patch *domain.PatchJobModel, jobId uuid.UUID) error {
	err := service.repo.PatchJob(ctx, patch, jobId)
	if err != nil {
		service.Logger.Error(
			"failed_patch_job",
			slog.Any("job_id", jobId),
			slog.Any("error", err),
		)
		return fmt.Errorf("error patching job by id %s", jobId)
	}
	service.Logger.Info(
		"success_patch_job",
		slog.Any("job_id", jobId),
	)
	return nil
}

func (service *ApiService) UpdateJobStatus(ctx context.Context, jobId uuid.UUID, status string) error {
	if err := service.repo.UpdateScheduleStatus(ctx, jobId, status); err != nil {
		service.Logger.Error(
			"failed_update_job_status",
			slog.Any("job_id", jobId),
			slog.String("new_status", status),
			slog.Any("error", err),
		)
		return err
	}
	service.Logger.Info(
		"success_update_job_status",
		slog.Any("job_id", jobId),
		slog.String("status", status),
	)
	return nil

}
