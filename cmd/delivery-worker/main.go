package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	generalConf "github.com/flamefks/scheduler-system/internal/config"
	coreConf "github.com/flamefks/scheduler-system/internal/delivery/config"
	"github.com/flamefks/scheduler-system/internal/delivery/service"
	logging "github.com/flamefks/scheduler-system/internal/logger"
	"github.com/flamefks/scheduler-system/internal/postgres"
	db "github.com/flamefks/scheduler-system/internal/postgres/queries"
	sharedData "github.com/flamefks/scheduler-system/internal/shared/data"
	qnats "github.com/flamefks/scheduler-system/internal/shared/queue/nats"
	sharedRepo "github.com/flamefks/scheduler-system/internal/shared/repository"
	"github.com/nats-io/nats.go"
)

func main() {
	appCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// logger config
	logCfg, err := generalConf.LoadLogging("config/logging.yml")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Logging config successfully parsed: %v", logCfg)

	// logger
	logger, err := logging.NewLogger(logCfg)
	if err != nil {
		log.Fatal(err)
	}
	logger.Info("logger_init", "status", "success")

	// core config
	coreCfg, err := coreConf.LoadAppConfig("config/core.yml")
	if err != nil {
		logger.Error("core_config_initialization",
			slog.String("status", "error"),
			slog.Any("err", err),
		)
		os.Exit(1)
	}
	log.Printf("Core config successfully parsed: %v", coreCfg)

	// Database
	pool, err := postgres.NewPool(appCtx, coreCfg.Postgres)
	if err != nil {
		logger.Error("postgres_connection",
			slog.String("status", "error"),
			slog.Any("err", err),
		)
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("postgres_connection",
		slog.String("status", "success"),
		slog.String("url", coreCfg.Postgres.DSN),
	)

	queries := db.New(pool)

	// nats
	nc, err := nats.Connect(coreCfg.Nats.Url)
	if err != nil {
		logger.Error("nats_connection",
			slog.String("status", "error"),
			slog.Any("err", err),
		)
		os.Exit(1)
	}
	defer nc.Drain()
	js := qnats.NewJetStream(appCtx, nc)

	logger.Info("nats_connection",
		slog.String("status", "success"),
		slog.String("url", coreCfg.Nats.Url),
	)

	// service initialization
	repository := sharedRepo.NewWorkerRepository(queries)
	schedulerService := service.NewDeliverService(
		logger,
		repository,
	)

	// consumer
	consumer := qnats.NewConsumer(js, sharedData.JobsSubjectDeliver)

	logger.Info("service_started")

	if err := consumer.Consume(appCtx, schedulerService.PipelineHandler, schedulerService.HandleError,
		sharedData.FetcherGroup); err != nil {
		logger.Error("consumer_stopped_with_error", slog.Any("err", err))
		os.Exit(1)
	}

	logger.Info("service_stopped")
}
