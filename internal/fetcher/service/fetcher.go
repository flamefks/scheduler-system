package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/flamefks/scheduler-system/internal/fetcher/client"
	"github.com/flamefks/scheduler-system/internal/fetcher/repository"
	natsqueue "github.com/flamefks/scheduler-system/internal/queue/nats"
	"github.com/flamefks/scheduler-system/internal/shared"
	"github.com/google/uuid"
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
		httpClient: shared.NewHTTPClient(),
		publisher:  publisher,
		repo:       repo,
	}
}

func (f *FetcherService) PipelineHandler(parentCtx context.Context, binJobId *[]byte) error {
	var jobId uuid.UUID
	err := jobId.UnmarshalBinary(*binJobId)
	if err != nil {
		f.logger.Error(
			"failed_unmarshal_id",
			slog.Any("bin_job_id", binJobId),
			slog.Any("err", err),
		)
		return err
	}

	ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
	defer cancel()

	reqConfig, err := f.repo.GetConfig(ctx, shared.FetcherKindName, jobId)
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

	request := &shared.Request{
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

	msgToNats := shared.NatsWorkerMessage{
		JobId:                jobId,
		ExternalResourceData: *response,
	}
	bytesMsg, err := json.Marshal(msgToNats)
	if err != nil {
		return err
	}

	err = f.publisher.Publish(ctx, "jobs.fetch", bytesMsg)
	if err != nil {
		f.logger.Error(
			"failed_publish_data",
			slog.Any("job_id", jobId),
			slog.Any("error", err),
		)
	}
	return nil
}

func (f *FetcherService) ErrorHandler(ctx context.Context, binJobId *[]byte) error {
	var jobId uuid.UUID
	err := jobId.UnmarshalBinary(*binJobId)
	if err != nil {
		f.logger.Error(
			"failed_unmarshal_id",
			slog.Any("bin_job_id", binJobId),
			slog.Any("err", err),
		)
		return err
	}

	return f.repo.SetJobStatus(ctx, "error", jobId)
}
