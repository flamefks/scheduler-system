package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/flamefks/scheduler-system/internal/api/domain"
	apimetrics "github.com/flamefks/scheduler-system/internal/api/metrics"
	"github.com/flamefks/scheduler-system/internal/shared/data"
	"github.com/google/uuid"

	repo "github.com/flamefks/scheduler-system/internal/api/repository"
)

type ApiService struct {
	Logger  *slog.Logger
	repo    repo.PostgresRepo
	metrics *apimetrics.ApiMetrics
}

func NewApiService(Logger *slog.Logger, r repo.PostgresRepo, metrics *apimetrics.ApiMetrics) *ApiService {
	return &ApiService{
		Logger:  Logger,
		repo:    r,
		metrics: metrics,
	}
}

func (service *ApiService) CreateJob(ctx context.Context, job *data.Job) (uuid.UUID, error) {
	jobId, err := service.repo.CreateJob(ctx, job)
	if err != nil {
		service.metrics.RecordDBOperation(ctx, apimetrics.OperationCreateJob, "error")
		service.Logger.Error(
			"create_job",
			slog.Any("job_data", job),
			slog.Any("error", err),
		)
		return uuid.Nil, err
	}
	service.metrics.RecordDBOperation(ctx, apimetrics.OperationCreateJob, "success")
	service.Logger.Info(
		"create_job",
		slog.Any("job_name", job.Name),
		slog.Any("job", &job),
	)
	return jobId, nil
}

func (service *ApiService) DeleteJob(ctx context.Context, jobId uuid.UUID) error {
	err := service.repo.DeleteJob(ctx, jobId)
	if err != nil {
		service.metrics.RecordDBOperation(ctx, apimetrics.OperationDeleteJob, "error")
		service.Logger.Error(
			"failed_delete_job",
			slog.Any("job_id", jobId),
			slog.Any("error", err),
		)
		return err
	}
	service.metrics.RecordDBOperation(ctx, apimetrics.OperationDeleteJob, "success")
	service.Logger.Info(
		"delete_job",
		slog.Any("job_id", jobId),
	)
	return nil
}

func (service *ApiService) GetJobByID(ctx context.Context, jobId uuid.UUID) (*data.Job, error) {
	j, err := service.repo.GetJobByID(ctx, jobId)
	if err != nil {
		service.metrics.RecordDBOperation(ctx, apimetrics.OperationGetJob, "error")
		service.Logger.Error(
			"get_job",
			slog.Any("job_id", jobId),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("error_get_job_by_id %s: %w", jobId, err)
	}
	service.metrics.RecordDBOperation(ctx, apimetrics.OperationGetJob, "success")
	service.Logger.Info(
		"get_job",
		slog.Any("job_id", jobId),
		slog.Any("job", &j),
	)
	return j, nil
}

func (service *ApiService) PatchJob(ctx context.Context, patch *domain.PatchJobModel, jobId uuid.UUID) error {
	err := service.repo.PatchJob(ctx, patch, jobId)
	if err != nil {
		service.metrics.RecordDBOperation(ctx, apimetrics.OperationPatchJob, "error")
		service.Logger.Error(
			"patch_job",
			slog.Any("job_id", jobId),
			slog.Any("error", err),
		)
		return fmt.Errorf("error_patch_job_by_id %s: %w", jobId, err)
	}
	service.metrics.RecordDBOperation(ctx, apimetrics.OperationPatchJob, "success")
	service.Logger.Info(
		"patch_job",
		slog.Any("job_id", jobId),
	)
	return nil
}

func (service *ApiService) ActivateJob(ctx context.Context, jobId uuid.UUID) error {
	if err := service.repo.ActivateJob(ctx, jobId); err != nil {
		service.metrics.RecordDBOperation(ctx, apimetrics.OperationActivateJob, "error")
		service.Logger.Error(
			"activate_job",
			slog.Any("job_id", jobId),
			slog.Any("error", err),
		)
		return fmt.Errorf("error_activate_job_by_id %s: %w", jobId, err)
	}
	service.metrics.RecordDBOperation(ctx, apimetrics.OperationActivateJob, "success")
	service.Logger.Info(
		"activate_job",
		slog.Any("job_id", jobId),
	)
	return nil
}

func (service *ApiService) DeactivateJob(ctx context.Context, jobId uuid.UUID) error {
	if err := service.repo.DeactivateJob(ctx, jobId); err != nil {
		service.metrics.RecordDBOperation(ctx, apimetrics.OperationDeactivateJob, "error")
		service.Logger.Error(
			"deactivate_job",
			slog.Any("job_id", jobId),
			slog.Any("error", err),
		)
		return fmt.Errorf("error_deactivate_job_by_id %s: %w", jobId, err)
	}
	service.metrics.RecordDBOperation(ctx, apimetrics.OperationDeactivateJob, "success")
	service.Logger.Info(
		"deactivate_job",
		slog.Any("job_id", jobId),
	)
	return nil
}
