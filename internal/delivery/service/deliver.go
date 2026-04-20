package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/flamefks/scheduler-system/internal/delivery/client"
	"github.com/flamefks/scheduler-system/internal/delivery/repository"
	ClientHttp "github.com/flamefks/scheduler-system/internal/shared/client/http"
	"github.com/flamefks/scheduler-system/internal/shared/data"
	"github.com/flamefks/scheduler-system/internal/shared/utils"
	"github.com/nats-io/nats.go"
)

type DeliverService struct {
	logger     *slog.Logger
	httpClient client.Client
	repo       repository.PostgresRepo
}

func NewDeliverService(logger *slog.Logger, repo repository.PostgresRepo) *DeliverService {
	return &DeliverService{
		logger:     logger,
		httpClient: ClientHttp.NewHTTPClient(),
		repo:       repo,
	}
}

func (ds *DeliverService) PipelineHandler(parentCtx context.Context, binNatsMsg []byte, natsHeader nats.Header) error {
	strJobId := natsHeader.Get("job-id")
	jobId, err := utils.GetJobIDFromHeader(strJobId)
	if err != nil {
		ds.logger.Error(
			"invalid_job_id_header",
			slog.String("job_id_raw", natsHeader.Get("job-id")),
			slog.Any("err", err),
		)
		return err
	}

	ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
	defer cancel()

	reqConfig, err := ds.repo.GetConfig(ctx, data.DeliverKindName, jobId)
	if err != nil {
		ds.logger.Error(
			"failed_get_config",
			slog.Any("job_id", jobId),
			slog.Any("err", err),
		)
		return err
	}

	var headerMap map[string]string
	if err := json.Unmarshal(reqConfig.HeaderAuth, &headerMap); err != nil {
		ds.logger.Error(
			"failed_unmarshal_config_header",
			slog.Any("job_id", jobId),
			slog.Any("err", err),
		)
		return err
	}

	request := &data.Request{
		Method:  reqConfig.Method,
		URL:     reqConfig.TargetUrl,
		Body:    binNatsMsg,
		Headers: headerMap,
	}

	response, err := ds.httpClient.Do(ctx, request)
	if err != nil {
		ds.logger.Error(
			"failed_http_request",
			slog.Any("job_id", jobId),
			slog.Any("err", err),
		)
		return err
	}
	if response.StatusCode != 200 {
		return fmt.Errorf("Got not 2xx code!")
	}

	return nil
}

func (f *DeliverService) ErrorHandler(ctx context.Context, binData []byte, natsHeader nats.Header) error {
	strJobId := natsHeader.Get("job-id")
	jobId, err := utils.GetJobIDFromHeader(strJobId)
	if err != nil {
		f.logger.Error(
			"invalid_job_id_header",
			slog.String("job_id_raw", natsHeader.Get("job-id")),
			slog.Any("err", err),
		)
		return err
	}

	return f.repo.SetJobStatus(ctx, "error", jobId)
}
