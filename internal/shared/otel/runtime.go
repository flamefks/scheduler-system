package otel

import (
	"context"
	"log/slog"
	"time"
)

func InitOrWarn(ctx context.Context, logger *slog.Logger, serviceName, serviceVersion, endpoint string) ShutdownFunc {
	shutdown, err := Init(ctx, serviceName, serviceVersion, endpoint)
	if err != nil {
		logger.Warn(
			"otel_init_failed",
			slog.Any("error", err),
		)
		return nil
	}

	logger.Info(
		"otel_init",
		slog.String("status", "success"),
		slog.String("endpoint", endpoint),
	)
	return shutdown
}

func ShutdownOrWarn(shutdown ShutdownFunc, logger *slog.Logger) {
	if shutdown == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := shutdown(ctx); err != nil {
		logger.Warn(
			"otel_shutdown_failed",
			slog.Any("error", err),
		)
	}
}
