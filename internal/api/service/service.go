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

func NewApiService(Logger *slog.Logger, r repo.PostgresRepo) *ApiService {
	apiMetrics, err := apimetrics.NewApiMetrics()
	if err != nil {
		Logger.Warn(
			"api_metrics_init_failed",
			slog.Any("error", err),
		)
	}

	return &ApiService{
		Logger:  Logger,
		repo:    r,
		metrics: apiMetrics,
	}
}

func (service *ApiService) CreateJob(ctx context.Context, job *data.Job) (uuid.UUID, error) {
	jobId, err := service.repo.CreateJob(ctx, job)
	if err != nil {
		service.metrics.Record(ctx, apimetrics.OperationCreateJob, err)
		service.Logger.Error(
			"create_job",
			slog.Any("job_data", job),
			slog.Any("error", err),
		)
		return uuid.Nil, err
	}
	service.metrics.Record(ctx, apimetrics.OperationCreateJob, nil)
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
		service.metrics.Record(ctx, apimetrics.OperationDeleteJob, err)
		service.Logger.Error(
			"failed_delete_job",
			slog.Any("job_id", jobId),
			slog.Any("error", err),
		)
		return err
	}
	service.metrics.Record(ctx, apimetrics.OperationDeleteJob, nil)
	service.Logger.Info(
		"delete_job",
		slog.Any("job_id", jobId),
	)
	return nil
}

func (service *ApiService) GetJobByID(ctx context.Context, jobId uuid.UUID) (*data.Job, error) {
	j, err := service.repo.GetJobByID(ctx, jobId)
	if err != nil {
		service.metrics.Record(ctx, apimetrics.OperationGetJob, err)
		service.Logger.Error(
			"get_job",
			slog.Any("job_id", jobId),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("error_get_job_by_id %s: %w", jobId, err)
	}
	service.metrics.Record(ctx, apimetrics.OperationGetJob, nil)
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
		service.metrics.Record(ctx, apimetrics.OperationPatchJob, err)
		service.Logger.Error(
			"patch_job",
			slog.Any("job_id", jobId),
			slog.Any("error", err),
		)
		return fmt.Errorf("error_patch_job_by_id %s: %w", jobId, err)
	}
	service.metrics.Record(ctx, apimetrics.OperationPatchJob, nil)
	service.Logger.Info(
		"patch_job",
		slog.Any("job_id", jobId),
	)
	return nil
}

func (service *ApiService) ActivateJob(ctx context.Context, jobId uuid.UUID) error {
	if err := service.repo.ActivateJob(ctx, jobId); err != nil {
		service.metrics.Record(ctx, apimetrics.OperationActivateJob, err)
		service.Logger.Error(
			"activate_job",
			slog.Any("job_id", jobId),
			slog.Any("error", err),
		)
		return fmt.Errorf("error_activate_job_by_id %s: %w", jobId, err)
	}
	service.metrics.Record(ctx, apimetrics.OperationActivateJob, nil)
	service.Logger.Info(
		"activate_job",
		slog.Any("job_id", jobId),
	)
	return nil
}

func (service *ApiService) DeactivateJob(ctx context.Context, jobId uuid.UUID) error {
	if err := service.repo.DeactivateJob(ctx, jobId); err != nil {
		service.metrics.Record(ctx, apimetrics.OperationDeactivateJob, err)
		service.Logger.Error(
			"deactivate_job",
			slog.Any("job_id", jobId),
			slog.Any("error", err),
		)
		return fmt.Errorf("error_deactivate_job_by_id %s: %w", jobId, err)
	}
	service.metrics.Record(ctx, apimetrics.OperationDeactivateJob, nil)
	service.Logger.Info(
		"deactivate_job",
		slog.Any("job_id", jobId),
	)
	return nil
}
