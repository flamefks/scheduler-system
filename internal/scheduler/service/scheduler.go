package service

import (
	"context"
	"log/slog"
	"time"

	repo "github.com/flamefks/scheduler-system/internal/scheduler/repository"
	qpublsher "github.com/flamefks/scheduler-system/internal/shared/queue/nats"
	"github.com/google/uuid"
)

type SchedulerService struct {
	logger    *slog.Logger
	repo      repo.PostgresRepo
	publisher *qpublsher.Publisher
}

func NewSchedulerService(logger *slog.Logger, r repo.PostgresRepo, p *qpublsher.Publisher) *SchedulerService {
	return &SchedulerService{
		logger:    logger,
		repo:      r,
		publisher: p,
	}
}

func (s *SchedulerService) ClaimNextJob(pctx context.Context) uuid.UUID {
	ctx, cancel := context.WithTimeout(pctx, 5*time.Second)
	defer cancel()

	id, err := s.repo.ClaimNextJob(ctx)
	if err != nil {
		s.logger.Error(
			"failed_claim_job",
			slog.Any("error", err),
		)
		return uuid.Nil
	}
	return id
}

func (s *SchedulerService) PublishJobIdToChannel(pctx context.Context, dataId uuid.UUID) {
	ctx, cancel := context.WithTimeout(pctx, 5*time.Second)
	defer cancel()

	natsHeaders := map[string]string{
		"job-id": dataId.String(),
	}

	err := s.publisher.Publish(ctx, "jobs.fetch", nil, natsHeaders)
	if err != nil {
		s.logger.Error(
			"failed_publish_uuid",
			slog.Any("error", err),
		)
	}
}

func (s *SchedulerService) MonitorTasksStatuses(parentCtx context.Context,
	JobDeathTimeout int64, pollInterval time.Duration) {
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
			err := s.repo.ResetHungMessage(dbCtx, JobDeathTimeout)
			stop()
			if err != nil {
				s.logger.Error("reset_hung_message", "status", "error", "msg", err)
			}
		}
	}
}
