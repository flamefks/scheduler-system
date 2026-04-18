package service

import (
	"log/slog"

	"github.com/flamefks/scheduler-system/internal/delivery/client"
	"github.com/flamefks/scheduler-system/internal/delivery/repository"
	"github.com/flamefks/scheduler-system/internal/shared"
)

type DeliverService struct {
	logger     *slog.Logger
	httpClient client.Client
	repo       repository.PostgresRepo
}

func NewDeliverService(logger *slog.Logger, repo repository.PostgresRepo) *DeliverService {
	return &DeliverService{
		logger:     logger,
		httpClient: shared.NewHTTPClient(),
		repo:       repo,
	}
}

func (ds *DeliverService) PipelineHandler() {

}
