package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/flamefks/scheduler-system/internal/api/domain"
	"github.com/flamefks/scheduler-system/internal/shared"
	"github.com/google/uuid"

	repo "github.com/flamefks/scheduler-system/internal/api/repository"
)

type ApiService struct {
	logger *slog.Logger
	repo   repo.PostgresRepo
}

func NewApiService(logger *slog.Logger, r repo.PostgresRepo) *ApiService {
	return &ApiService{
		logger: logger,
		repo:   r,
	}
}

func (service *ApiService) CreateJob(ctx context.Context, job *shared.Job) (uuid.UUID, error) {
	jobId, err := service.repo.CreateJob(ctx, job)
	if err != nil {
		service.logger.Error(
			"failed_create_job",
			slog.Any("job_data", job),
			slog.Any("error", err),
		)
		return uuid.Nil, fmt.Errorf("Error on creating job: %w", err)
	}
	return jobId, nil
}

func (service *ApiService) DeleteJob(ctx context.Context, jobId uuid.UUID) error {
	err := service.repo.DeleteJob(ctx, jobId)
	if err != nil {
		service.logger.Error(
			"failed_delete_job",
			slog.Any("job_id", jobId),
			slog.Any("error", err),
		)
		return fmt.Errorf("Error on removing job: %w", err)
	}
	return nil
}

func (service *ApiService) GetJobByID(ctx context.Context, jobId uuid.UUID) (*shared.Job, error) {
	j, err := service.repo.GetJobByID(ctx, jobId)
	if err != nil {
		service.logger.Error(
			"failed_get_job",
			slog.Any("job_id", jobId),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("error getting job by id %s", jobId)
	}
	return j, nil
}

func (service *ApiService) PatchJob(ctx context.Context, patch *domain.PatchJobModel, jobId uuid.UUID) error {
	err := service.repo.PatchJob(ctx, patch, jobId)
	if err != nil {
		service.logger.Error(
			"failed_patch_job",
			slog.Any("job_id", jobId),
			slog.Any("error", err),
		)
		return fmt.Errorf("error patching job by id %s", jobId)
	}
	return nil
}
