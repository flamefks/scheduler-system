package service

import (
	"context"
	"log/slog"
	"time"

	schedulermetrics "github.com/flamefks/scheduler-system/internal/scheduler/metrics"
	repo "github.com/flamefks/scheduler-system/internal/scheduler/repository"
	sharedData "github.com/flamefks/scheduler-system/internal/shared/data"
	qpublsher "github.com/flamefks/scheduler-system/internal/shared/queue/nats"
	"github.com/google/uuid"
)

type SchedulerService struct {
	logger    *slog.Logger
	repo      repo.PostgresRepo
	publisher qpublsher.AbstractPublisher
	metrics   *schedulermetrics.SchedulerMetrics
}

func NewSchedulerService(logger *slog.Logger, r repo.PostgresRepo, p qpublsher.AbstractPublisher, metrics *schedulermetrics.SchedulerMetrics) *SchedulerService {
	return &SchedulerService{
		logger:    logger,
		repo:      r,
		publisher: p,
		metrics:   metrics,
	}
}

func (s *SchedulerService) ClaimNextJobs(pctx context.Context, jobBatchSize int) []uuid.UUID {
	ctx, cancel := context.WithTimeout(pctx, 5*time.Second)
	defer cancel()

	idList, err := s.repo.ClaimNextJobs(ctx, jobBatchSize)
	if err != nil {
		s.logger.Error(
			"failed_claim_jobs",
			slog.Any("error", err),
		)
		s.metrics.RecordClaimed(ctx, "error")
		return nil
	}

	s.logger.Info(
		"success_claim_jobs",
		slog.Int("jobs_count", len(idList)),
		slog.Any("job_id_list", idList),
	)
	s.metrics.RecordClaimed(ctx, "success")
	s.metrics.RecordClaimedJobs(ctx, len(idList))

	return idList
}

func (s *SchedulerService) PublishJobIdToChannel(pctx context.Context, dataId uuid.UUID) {
	ctx, cancel := context.WithTimeout(pctx, 5*time.Second)
	defer cancel()

	natsHeaders := map[string]string{
		"job-id": dataId.String(),
	}

	err := s.publisher.Publish(ctx, sharedData.JobsSubjectFetcher, nil, natsHeaders)
	if err != nil {
		s.metrics.RecordNatsPublish(ctx, "error")
		s.logger.Error(
			"failed_publish_uuid",
			slog.Any("error", err),
		)
		return
	}
	s.metrics.RecordNatsPublish(ctx, "success")
	s.logger.Info(
		"success_publish_job_id",
		slog.String("job_id", dataId.String()),
	)
}

func (s *SchedulerService) MonitorHungedTasks(parentCtx context.Context,
	scheduleJobTimeout int, procJobTimeout int, pollInterval time.Duration) {
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dbCtx, stop := context.WithTimeout(ctx, 5*time.Second)
			rowsAffected, err := s.repo.ResetHungMessage(dbCtx, scheduleJobTimeout, procJobTimeout)
			if err != nil {
				s.metrics.RecordResetHung(dbCtx, "error")
				stop()
				s.logger.Error(
					"error_reset_hung_message",
					slog.String("status", "error"),
					slog.Any("msg", err),
				)
				continue
			}
			s.metrics.RecordResetHung(dbCtx, "success")
			s.metrics.RecordResetHungJobs(dbCtx, rowsAffected)
			stop()
		}
	}
}

func (s *SchedulerService) MonitorDisabledTasks(parentCtx context.Context, pollInterval time.Duration) {
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dbCtx, stop := context.WithTimeout(ctx, 5*time.Second)
			rowsAffected, err := s.repo.SwitchToDisabledIfNeed(dbCtx)
			if err != nil {
				s.metrics.RecordDisabled(dbCtx, "error")
				stop()
				s.logger.Error(
					"error_disable_task",
					slog.String("status", "error"),
					slog.Any("msg", err),
				)
				continue
			}
			s.metrics.RecordDisabled(dbCtx, "success")
			s.metrics.RecordDisabledJobs(dbCtx, rowsAffected)
			stop()
		}
	}
}
