package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	repo "github.com/flamefks/scheduler-system/internal/scheduler/repository"
	qpublsher "github.com/flamefks/scheduler-system/internal/shared/queue/nats"
	"github.com/google/uuid"
)

type SchedulerService struct {
	logger    *slog.Logger
	repo      repo.PostgresRepo
	publisher qpublsher.AbstractPublisher
}

func NewSchedulerService(logger *slog.Logger, r repo.PostgresRepo, p qpublsher.AbstractPublisher) *SchedulerService {
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
		if !errors.Is(err, sql.ErrNoRows) {
			s.logger.Error(
				"failed_claim_job",
				slog.Any("error", err),
			)
		}

		return uuid.Nil
	}

	s.logger.Info(
		"success_claim_job",
		slog.Any("job_id", id),
	)
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

func (s *SchedulerService) MonitorHungedTasks(parentCtx context.Context,
	JobDeathTimeout int, pollInterval time.Duration) {
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
				s.logger.Error(
					"reset_hung_message",
					slog.String("status", "error"),
					slog.Any("msg", err),
				)
			}
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
			err := s.repo.SwitchToDisabledIfNeed(dbCtx)
			stop()
			if err != nil {
				s.logger.Error(
					"reset_hung_message",
					slog.String("status", "error"),
					slog.Any("msg", err),
				)
			}
		}
	}
}
