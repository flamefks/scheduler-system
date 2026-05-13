package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/flamefks/scheduler-system/internal/fetcher/client"
	coreConf "github.com/flamefks/scheduler-system/internal/fetcher/config"
	"github.com/flamefks/scheduler-system/internal/fetcher/repository"
	ClientHttp "github.com/flamefks/scheduler-system/internal/shared/client/http"
	"github.com/flamefks/scheduler-system/internal/shared/data"
	sharedData "github.com/flamefks/scheduler-system/internal/shared/data"
	natsqueue "github.com/flamefks/scheduler-system/internal/shared/queue/nats"
	"github.com/flamefks/scheduler-system/internal/shared/utils"
	"github.com/nats-io/nats.go"
)

type FetcherService struct {
	logger     *slog.Logger
	httpClient client.Client
	publisher  natsqueue.AbstractPublisher
	repo       repository.PostgresRepo
}

func NewFetcherService(logger *slog.Logger, publisher natsqueue.AbstractPublisher, repo repository.PostgresRepo) *FetcherService {
	return &FetcherService{
		logger:     logger,
		httpClient: ClientHttp.NewHTTPClient(),
		publisher:  publisher,
		repo:       repo,
	}
}

func (f *FetcherService) Handle(parentCtx context.Context, binData []byte, natsHeader nats.Header, needSetDbStatus *bool) (error, int) {
	strJobId := natsHeader.Get("job-id")
	jobId, err := natsqueue.GetJobIDFromHeader(strJobId)
	if err != nil {
		f.logger.Error(
			"invalid_job_id_header",
			slog.String("job_id_raw", natsHeader.Get("job-id")),
			slog.Any("err", err),
		)
		return natsqueue.TermError, 0
	}

	ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
	defer cancel()

	if *needSetDbStatus {
		err = f.repo.SetJobStatus(ctx, "fetching", jobId)
		if err != nil {
			f.logger.Error(
				"failed_set_job_status",
				slog.Any("job_id", jobId),
				slog.String("new_status", "fetching"),
				slog.Any("err", err),
			)
			return natsqueue.NakError, 0
		}
	}

	*needSetDbStatus = false

	reqConfig, err := f.repo.GetConfig(ctx, data.FetcherKindName, jobId)
	if err != nil {
		f.logger.Error(
			"failed_get_config",
			slog.Any("job_id", jobId),
			slog.Any("err", err),
		)
		return natsqueue.NakError, 0
	}
	f.logger.Info(
		"success_get_config",
		slog.String("job_id", strJobId),
		slog.Any("config", &reqConfig),
	)

	headerMap := map[string]string{}
	if len(reqConfig.Headers) > 0 {
		if err := json.Unmarshal(reqConfig.Headers, &headerMap); err != nil {
			f.logger.Error(
				"failed_unmarshal_headers",
				slog.Any("job_id", jobId),
				slog.Any("err", err),
			)
			return natsqueue.TermError, 0
		}
	}

	request := &data.Request{
		Method:  reqConfig.Method,
		URL:     reqConfig.TargetUrl,
		Body:    reqConfig.Payload,
		Headers: headerMap,
	}

	response, err := f.httpClient.Do(ctx, request)
	if err != nil {
		statusCode := 0
		if response != nil {
			statusCode = response.StatusCode
		}

		f.logger.Error(
			"failed_http_request",
			slog.Any("job_id", jobId),
			slog.Any("err", err),
		)
		return natsqueue.NakError, statusCode
	}
	f.logger.Info(
		"response",
		slog.String("job_id", strJobId),
		slog.Any("data", &response),
	)

	if len(reqConfig.JsonSchema) > 0 {
		if err = utils.ValidateRawMessageWithSchema(reqConfig.JsonSchema, response.Body); err != nil {
			f.logger.Error(
				"failed_validate_schema",
				slog.Any("job_id", jobId),
				slog.Any("schema", reqConfig.JsonSchema),
				slog.Any("response", response.Body),
				slog.Any("err", err),
			)
			return natsqueue.TermError, 0
		}
	}

	err = f.publisher.Publish(ctx, sharedData.JobsSubjectDeliver, response.Body, map[string]string{
		"job-id": strJobId,
	})

	if err != nil {
		f.logger.Error(
			"failed_publish_data",
			slog.Any("job_id", jobId),
			slog.Any("error", err),
		)
		return natsqueue.NakError, 0
	}
	return nil, response.StatusCode
}

func (f *FetcherService) ErrorHandler(ctx context.Context, binData []byte, natsHeader nats.Header) {
	strJobId := natsHeader.Get("job-id")
	jobId, err := natsqueue.GetJobIDFromHeader(strJobId)
	if err != nil {
		f.logger.Error(
			"invalid_job_id_header",
			slog.String("job_id_raw", natsHeader.Get("job-id")),
			slog.Any("err", err),
		)
		return
	}

	err = f.repo.SetJobStatus(ctx, "error", jobId)

	if err != nil {
		f.logger.Error(
			"failed_set_job_error",
			slog.Any("err", err),
		)
		return
	}
	f.logger.Info(
		"success_handle_error",
		slog.String("job_id", strJobId),
	)
}

func (f *FetcherService) PipelineHandler(parentCtx context.Context, binData []byte, natsHeader nats.Header) error {
	config := coreConf.GetCoreConfig().HttpRetry
	needNotifyDb := true
	delay := config.BaseDelay
	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		select {
		case <-parentCtx.Done():
			return parentCtx.Err()
		default:
		}

		err, statusCode := f.Handle(parentCtx, binData, natsHeader, &needNotifyDb)
		isHttpError := utils.InSlice(config.RetryOnStatus, statusCode)
		if !isHttpError {
			return err
		}

		f.logger.Warn(
			"pipeline_handler_failed",
			slog.Int("attempt", attempt),
			slog.Int("http_status_code", statusCode),
			slog.Any("error", err),
		)

		if attempt == config.MaxAttempts-1 {
			if err == nil && isHttpError {
				return fmt.Errorf("Http_status_code_error: %d; err = %w", statusCode, natsqueue.NakError)
			} else {
				return err
			}
		}

		if config.Backoff == "exponential" {
			delay = utils.BackoffDuration(attempt, config.BaseDelay, config.MaxDelay)
		}
		f.logger.Debug("waiting before retry", slog.Duration("delay", delay))

		select {
		case <-time.After(delay):
		case <-parentCtx.Done():
			return parentCtx.Err()
		}
	}

	return fmt.Errorf("max retries exceeded")
}
