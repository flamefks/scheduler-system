package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/flamefks/scheduler-system/internal/fetcher/client"
	coreConf "github.com/flamefks/scheduler-system/internal/fetcher/config"
	fetchermetrics "github.com/flamefks/scheduler-system/internal/fetcher/metrics"
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
	metrics    *fetchermetrics.FetcherMetrics
}

func NewFetcherService(logger *slog.Logger, publisher natsqueue.AbstractPublisher, repo repository.PostgresRepo, metrics *fetchermetrics.FetcherMetrics) *FetcherService {
	return &FetcherService{
		logger:     logger,
		httpClient: ClientHttp.NewHTTPClient(),
		publisher:  publisher,
		repo:       repo,
		metrics:    metrics,
	}
}

func (f *FetcherService) Handle(parentCtx context.Context, binData []byte, natsHeader nats.Header, needSetDbStatus *bool, retryOnStatus []int) (error, int) {
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
	strRunId := natsHeader.Get("run-id")
	runId, err := natsqueue.GetRunIDFromHeader(strRunId)
	if err != nil {
		f.logger.Error(
			"invalid_run_id_header",
			slog.String("run_id_raw", strRunId),
			slog.Any("err", err),
		)
		return natsqueue.TermError, 0
	}

	ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
	defer cancel()

	if *needSetDbStatus {
		err = f.repo.SetJobRunStatus(ctx, "fetching", jobId, runId)
		if err != nil {
			f.logger.Error(
				"failed_set_job_run_status",
				slog.Any("job_id", jobId),
				slog.Any("run_id", runId),
				slog.String("new_status", "fetching"),
				slog.Any("err", err),
			)
			return natsqueue.TermError, 0
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
		f.metrics.RecordHTTPRequest(ctx, "error", statusCode)
		return natsqueue.NakError, statusCode
	}
	isRetryableStatus := utils.InSlice(retryOnStatus, response.StatusCode)
	if isRetryableStatus {
		f.metrics.RecordHTTPRequest(ctx, "error", response.StatusCode)
		f.logger.Warn(
			"retryable_http_status",
			slog.Any("job_id", jobId),
			slog.Int("http_status_code", response.StatusCode),
		)
		return natsqueue.NakError, response.StatusCode
	}

	f.metrics.RecordHTTPRequest(ctx, "success", response.StatusCode)
	f.logger.Info(
		"fetcher_http_response",
		slog.String("job_id", strJobId),
		slog.String("run_id", strRunId),
		slog.Int("http_status_code", response.StatusCode),
		slog.Int("response_body_bytes", len(response.Body)),
	)

	if len(reqConfig.JsonSchema) > 0 {
		if err = utils.ValidateRawMessageWithSchema(reqConfig.JsonSchema, response.Body); err != nil {
			f.logger.Error(
				"failed_validate_schema",
				slog.Any("job_id", jobId),
				slog.Any("err", err),
			)
			return natsqueue.TermError, 0
		}
	}

	err = f.publisher.Publish(ctx, sharedData.JobsSubjectDeliver, response.Body, map[string]string{
		"job-id": strJobId,
		"run-id": strRunId,
	})

	if err != nil {
		f.metrics.RecordNatsPublish(ctx, "error")
		f.logger.Error(
			"failed_publish_data",
			slog.Any("job_id", jobId),
			slog.Any("error", err),
		)
		return natsqueue.NakError, 0
	}
	f.metrics.RecordNatsPublish(ctx, "success")
	f.metrics.RecordNatsPublishJobs(ctx, 1)
	return nil, response.StatusCode
}

func (f *FetcherService) ErrorHandler(ctx context.Context, binData []byte, natsHeader nats.Header) error {
	strJobId := natsHeader.Get("job-id")
	jobId, err := natsqueue.GetJobIDFromHeader(strJobId)
	if err != nil {
		f.metrics.RecordErrorHandler(ctx, "error")
		f.logger.Error(
			"invalid_job_id_header",
			slog.String("job_id_raw", natsHeader.Get("job-id")),
			slog.Any("err", err),
		)
		return err
	}
	strRunId := natsHeader.Get("run-id")
	runId, err := natsqueue.GetRunIDFromHeader(strRunId)
	if err != nil {
		f.metrics.RecordErrorHandler(ctx, "error")
		f.logger.Error(
			"invalid_run_id_header",
			slog.String("run_id_raw", strRunId),
			slog.Any("err", err),
		)
		return err
	}

	err = f.repo.SetJobRunStatus(ctx, "error", jobId, runId)

	if err != nil {
		f.metrics.RecordErrorHandler(ctx, "error")
		f.logger.Error(
			"failed_set_job_error",
			slog.Any("err", err),
		)
		return err
	}
	f.metrics.RecordErrorHandler(ctx, "success")
	f.metrics.RecordErrorHandlerJobs(ctx, 1)
	f.logger.Info(
		"success_handle_error",
		slog.String("job_id", strJobId),
	)

	return nil
}

func (f *FetcherService) PipelineHandler(
	parentCtx context.Context, binData []byte, natsHeader nats.Header, deliveryAttempt uint64,
	maxTimeCompleting time.Duration,
) error {
	config := coreConf.GetCoreConfig().HttpRetry
	needNotifyDb := true
	delay := config.BaseDelay
	deadline := time.Now().Add(maxTimeCompleting)

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		select {
		case <-parentCtx.Done():
			return parentCtx.Err()
		default:
		}

		err, statusCode := f.Handle(parentCtx, binData, natsHeader, &needNotifyDb, config.RetryOnStatus)
		isHttpError := utils.InSlice(config.RetryOnStatus, statusCode)
		if !isHttpError {
			return err
		}

		f.logger.Warn(
			"pipeline_handler_failed",
			slog.Uint64("delivery_attempt", deliveryAttempt),
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
		var ok bool
		delay, ok = fitRetryDelay(delay, deadline)
		if !ok {
			f.logger.Warn(
				"http_retry_budget_exhausted",
				slog.Uint64("delivery_attempt", deliveryAttempt),
				slog.Int("attempt", attempt),
				slog.Int("http_status_code", statusCode),
				slog.Duration("max_time_completing", maxTimeCompleting),
				slog.Any("error", err),
			)
			return err
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

func fitRetryDelay(delay time.Duration, deadline time.Time) (time.Duration, bool) {
	const ackSafetyGap = 10 * time.Second

	remaining := time.Until(deadline) - ackSafetyGap
	if remaining <= 0 {
		return 0, false
	}
	if delay > remaining {
		return remaining, true
	}
	return delay, true
}
