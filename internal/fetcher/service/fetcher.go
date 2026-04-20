package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/flamefks/scheduler-system/internal/fetcher/client"
	"github.com/flamefks/scheduler-system/internal/fetcher/repository"
	ClientHttp "github.com/flamefks/scheduler-system/internal/shared/client/http"
	"github.com/flamefks/scheduler-system/internal/shared/data"
	natsqueue "github.com/flamefks/scheduler-system/internal/shared/queue/nats"
	"github.com/flamefks/scheduler-system/internal/shared/utils"
	"github.com/nats-io/nats.go"
)

type FetcherService struct {
	logger     *slog.Logger
	httpClient client.Client
	publisher  *natsqueue.Publisher
	repo       repository.PostgresRepo
}

func NewFetcherService(logger *slog.Logger, publisher *natsqueue.Publisher, repo repository.PostgresRepo) *FetcherService {
	return &FetcherService{
		logger:     logger,
		httpClient: ClientHttp.NewHTTPClient(),
		publisher:  publisher,
		repo:       repo,
	}
}

func (f *FetcherService) PipelineHandler(parentCtx context.Context, binData []byte, natsHeader nats.Header) error {
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

	ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
	defer cancel()

	reqConfig, err := f.repo.GetConfig(ctx, data.FetcherKindName, jobId)
	if err != nil {
		f.logger.Error(
			"failed_get_config",
			slog.Any("job_id", jobId),
			slog.Any("err", err),
		)
		return err
	}

	var headerMap map[string]string
	if err := json.Unmarshal(reqConfig.HeaderAuth, &headerMap); err != nil {
		f.logger.Error(
			"failed_unmarshal_header_auth",
			slog.Any("job_id", jobId),
			slog.Any("err", err),
		)
		return err
	}

	request := &data.Request{
		Method:  reqConfig.Method,
		URL:     reqConfig.TargetUrl,
		Body:    reqConfig.Payload,
		Headers: headerMap,
	}

	response, err := f.httpClient.Do(ctx, request)
	if err != nil {
		f.logger.Error(
			"failed_http_request",
			slog.Any("job_id", jobId),
			slog.Any("err", err),
		)
		return err
	}

	bytesMsg, err := json.Marshal(response)
	if err != nil {
		return err
	}

	err = f.publisher.Publish(ctx, "jobs.fetch", bytesMsg, map[string]string{
		"job-id": strJobId,
	})
	if err != nil {
		f.logger.Error(
			"failed_publish_data",
			slog.Any("job_id", jobId),
			slog.Any("error", err),
		)
	}
	return nil
}

func (f *FetcherService) ErrorHandler(ctx context.Context, binData []byte, natsHeader nats.Header) error {
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
