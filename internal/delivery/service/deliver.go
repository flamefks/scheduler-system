package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/flamefks/scheduler-system/internal/delivery/client"
	coreConf "github.com/flamefks/scheduler-system/internal/delivery/config"
	deliverymetrics "github.com/flamefks/scheduler-system/internal/delivery/metrics"
	"github.com/flamefks/scheduler-system/internal/delivery/repository"
	ClientHttp "github.com/flamefks/scheduler-system/internal/shared/client/http"
	"github.com/flamefks/scheduler-system/internal/shared/data"
	natsqueue "github.com/flamefks/scheduler-system/internal/shared/queue/nats"
	"github.com/flamefks/scheduler-system/internal/shared/utils"
	"github.com/nats-io/nats.go"
)

type DeliverService struct {
	logger     *slog.Logger
	httpClient client.Client
	repo       repository.PostgresRepo
	metrics    *deliverymetrics.DeliveryMetrics
}

func NewDeliverService(logger *slog.Logger, repo repository.PostgresRepo, metrics *deliverymetrics.DeliveryMetrics) *DeliverService {
	return &DeliverService{
		logger:     logger,
		httpClient: ClientHttp.NewHTTPClient(),
		repo:       repo,
		metrics:    metrics,
	}
}

func (ds *DeliverService) Handle(parentCtx context.Context, binNatsMsg []byte, natsHeader nats.Header, needSetDbStatus *bool, retryOnStatus []int) (error, int) {

	strJobId := natsHeader.Get("job-id")
	jobId, err := natsqueue.GetJobIDFromHeader(strJobId)
	if err != nil {
		ds.logger.Error(
			"invalid_job_id_header",
			slog.String("job_id_raw", natsHeader.Get("job-id")),
			slog.Any("err", err),
		)
		return natsqueue.TermError, 0
	}
	strRunId := natsHeader.Get("run-id")
	runId, err := natsqueue.GetRunIDFromHeader(strRunId)
	if err != nil {
		ds.logger.Error(
			"invalid_run_id_header",
			slog.String("run_id_raw", strRunId),
			slog.Any("err", err),
		)
		return natsqueue.TermError, 0
	}

	ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
	defer cancel()

	if *needSetDbStatus {
		err = ds.repo.SetJobRunStatus(ctx, "delivering", jobId, runId)
		if err != nil {
			ds.logger.Error(
				"failed_set_job_run_status",
				slog.Any("job_id", jobId),
				slog.Any("run_id", runId),
				slog.String("new_status", "delivering"),
				slog.Any("err", err),
			)
			return natsqueue.TermError, 0
		}
	}
	*needSetDbStatus = false

	reqConfig, err := ds.repo.GetConfig(ctx, data.DeliverKindName, jobId)
	if err != nil {
		ds.logger.Error(
			"failed_get_config",
			slog.Any("job_id", jobId),
			slog.Any("err", err),
		)
		return natsqueue.NakError, 0
	}
	ds.logger.Info(
		"success_get_config",
		slog.String("job_id", strJobId),
	)

	headerMap := map[string]string{}
	if len(reqConfig.Headers) > 0 {
		if err := json.Unmarshal(reqConfig.Headers, &headerMap); err != nil {
			ds.logger.Error(
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
		Body:    binNatsMsg,
		Headers: headerMap,
	}

	response, err := ds.httpClient.Do(ctx, request)
	if err != nil {
		statusCode := 0
		if response != nil {
			statusCode = response.StatusCode
		}

		ds.logger.Error(
			"failed_http_request",
			slog.Any("job_id", jobId),
			slog.Any("err", err),
		)
		ds.metrics.RecordHTTPRequest(ctx, "error", statusCode)
		return natsqueue.NakError, statusCode
	}
	isRetryableStatus := utils.InSlice(retryOnStatus, response.StatusCode)
	if isRetryableStatus {
		ds.metrics.RecordHTTPRequest(ctx, "error", response.StatusCode)
		ds.logger.Warn(
			"retryable_http_status",
			slog.Any("job_id", jobId),
			slog.Int("http_status_code", response.StatusCode),
		)
		return natsqueue.NakError, response.StatusCode
	}

	ds.metrics.RecordHTTPRequest(ctx, "success", response.StatusCode)
	ds.logger.Info(
		"success_sent_response",
		slog.String("job_id", strJobId),
		slog.String("run_id", strRunId),
		slog.Int("http_status_code", response.StatusCode),
		slog.Int("response_body_bytes", len(response.Body)),
	)

	err = ds.repo.SetJobRunStatus(ctx, "idle", jobId, runId)
	if err != nil {
		return natsqueue.TermError, 0
	}
	ds.metrics.RecordDeliveryJobs(ctx, 1)
	return nil, response.StatusCode
}

func (ds *DeliverService) HandleError(ctx context.Context, binData []byte, natsHeader nats.Header) {
	strJobId := natsHeader.Get("job-id")
	jobId, err := natsqueue.GetJobIDFromHeader(strJobId)
	if err != nil {
		ds.metrics.RecordErrorHandler(ctx, "error")
		ds.logger.Error(
			"invalid_job_id_header",
			slog.String("job_id_raw", natsHeader.Get("job-id")),
			slog.Any("err", err),
		)
		return
	}
	strRunId := natsHeader.Get("run-id")
	runId, err := natsqueue.GetRunIDFromHeader(strRunId)
	if err != nil {
		ds.metrics.RecordErrorHandler(ctx, "error")
		ds.logger.Error(
			"invalid_run_id_header",
			slog.String("run_id_raw", strRunId),
			slog.Any("err", err),
		)
		return
	}

	err = ds.repo.SetJobRunStatus(ctx, "error", jobId, runId)
	if err != nil {
		ds.metrics.RecordErrorHandler(ctx, "error")
		ds.logger.Error(
			"failed_set_job_error",
			slog.Any("err", err),
		)
		return
	}
	ds.metrics.RecordErrorHandler(ctx, "success")
	ds.metrics.RecordErrorHandlerJobs(ctx, 1)
	ds.logger.Info(
		"success_handle_error",
		slog.String("job_id", strJobId),
	)
}

func (ds *DeliverService) PipelineHandler(parentCtx context.Context, binNatsMsg []byte, natsHeader nats.Header, deliveryAttempt uint64) error {
	config := coreConf.GetCoreConfig().HttpRetry
	needNotifyDb := true
	delay := config.BaseDelay
	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		select {
		case <-parentCtx.Done():
			return parentCtx.Err()
		default:
		}

		err, statusCode := ds.Handle(parentCtx, binNatsMsg, natsHeader, &needNotifyDb, config.RetryOnStatus)
		isHttpError := utils.InSlice(config.RetryOnStatus, statusCode)
		if err == nil && !isHttpError {
			return nil
		}

		ds.logger.Warn(
			"pipeline_handler_failed",
			slog.Uint64("delivery_attempt", deliveryAttempt),
			slog.Int("attempt", attempt),
			slog.Int("http_status_code", statusCode),
			slog.Any("error", err),
		)

		if attempt == config.MaxAttempts-1 {
			if err == nil && isHttpError {
				return fmt.Errorf("Http_status_code_error: %d", statusCode)
			} else {
				return err
			}
		}

		if config.Backoff == "exponential" {
			delay = utils.BackoffDuration(attempt, config.BaseDelay, config.MaxDelay)
		}
		ds.logger.Debug("waiting before retry", slog.Duration("delay", delay))

		select {
		case <-time.After(delay):
		case <-parentCtx.Done():
			return parentCtx.Err()
		}
	}

	return fmt.Errorf("max_retries_exceeded")
}
