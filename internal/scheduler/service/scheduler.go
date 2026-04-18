package service

import (
	"context"
	"log/slog"
	"time"

	qpublsher "github.com/flamefks/scheduler-system/internal/queue/nats"
	repo "github.com/flamefks/scheduler-system/internal/scheduler/repository"
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
	binId, err := dataId.MarshalBinary()
	if err != nil {
		s.logger.Error(
			"failed_marshal_uuid",
			slog.Any("error", err),
		)
	}

	ctx, cancel := context.WithTimeout(pctx, 5*time.Second)
	defer cancel()

	err = s.publisher.Publish(ctx, "jobs.fetch", binId)
	if err != nil {
		s.logger.Error(
			"failed_publish_uuid",
			slog.Any("error", err),
		)
	}
}

// мониторит статусы (служит для зависших active / error задач, для их перевода в idle)
func (s *SchedulerService) MonitorTasksStatuses(parentCtx context.Context) {

}
