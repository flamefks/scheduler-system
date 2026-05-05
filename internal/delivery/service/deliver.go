package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/flamefks/scheduler-system/internal/delivery/client"
	coreConf "github.com/flamefks/scheduler-system/internal/delivery/config"
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

func (ds *DeliverService) Handle(parentCtx context.Context, binNatsMsg []byte, natsHeader nats.Header) (error, int) {

	strJobId := natsHeader.Get("job-id")
	jobId, err := utils.GetJobIDFromHeader(strJobId)
	if err != nil {
		ds.logger.Error(
			"invalid_job_id_header",
			slog.String("job_id_raw", natsHeader.Get("job-id")),
			slog.Any("err", err),
		)
		return err, 0
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
		return err, 0
	}
	ds.logger.Info(
		"success_get_config",
		slog.String("job_id", strJobId),
		slog.Any("config", &reqConfig),
	)

	var headerMap map[string]string
	if err := json.Unmarshal(reqConfig.Headers, &headerMap); err != nil {
		ds.logger.Error(
			"failed_unmarshal_config_header",
			slog.Any("job_id", jobId),
			slog.Any("err", err),
		)
		return err, 0
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
		return err, statusCode
	}
	ds.logger.Info(
		"success_sent_reponse",
		slog.String("job_id", strJobId),
		slog.Any("data", &response),
	)

	err = ds.repo.SetJobStatus(ctx, "idle", jobId)
	if err != nil {
		return err, 0
	}
	return nil, response.StatusCode
}

func (ds *DeliverService) HandleError(ctx context.Context, binData []byte, natsHeader nats.Header) error {
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

	err = ds.repo.SetJobStatus(ctx, "error", jobId)
	if err != nil {
		ds.logger.Error(
			"failed_set_job_error",
			slog.Any("err", err),
		)
		return err
	}
	ds.logger.Info(
		"success_handle_error",
		slog.String("job_id", strJobId),
	)
	return nil
}

func (ds *DeliverService) PipelineHandler(parentCtx context.Context, binNatsMsg []byte, natsHeader nats.Header) error {
	config := coreConf.GetCoreConfig().HttpRetry

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		select {
		case <-parentCtx.Done():
			return parentCtx.Err()
		default:
		}

		err, statusCode := ds.Handle(parentCtx, binNatsMsg, natsHeader)
		if err == nil && !utils.InSlice(config.RetryOnStatus, statusCode) {
			return nil
		}

		ds.logger.Warn(
			"pipeline_handler_failed",
			slog.Int("attempt", attempt),
			slog.Int("http_status_code", statusCode),
			slog.Any("error", err),
		)

		if !config.RetryOnError || attempt == config.MaxAttempts-1 {
			return err
		}

		delay := utils.BackoffDuration(attempt, config.BaseDelay, config.MaxDelay)
		ds.logger.Debug("waiting before retry", slog.Duration("delay", delay))

		select {
		case <-time.After(delay):
		case <-parentCtx.Done():
			return parentCtx.Err()
		}
	}

	return fmt.Errorf("max retries exceeded")
}
