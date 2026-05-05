package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	generalConf "github.com/flamefks/scheduler-system/internal/config"
	logging "github.com/flamefks/scheduler-system/internal/logger"
	"github.com/flamefks/scheduler-system/internal/postgres"
	db "github.com/flamefks/scheduler-system/internal/postgres/queries"
	coreConf "github.com/flamefks/scheduler-system/internal/scheduler/config"
	repo "github.com/flamefks/scheduler-system/internal/scheduler/repository"
	service "github.com/flamefks/scheduler-system/internal/scheduler/service"
	qnats "github.com/flamefks/scheduler-system/internal/shared/queue/nats"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"gopkg.in/yaml.v3"
)

func main() {
	appCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// logger config
	logCfg, err := generalConf.LoadLogging("config/logging.yml")
	if err != nil {
		log.Fatal(err)
	}
	b, err := yaml.Marshal(logCfg)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Logging config successfully parsed: %v", string(b))

	// logger
	logger, err := logging.NewLogger(logCfg)
	if err != nil {
		log.Fatal(err)
	}
	logger.Info("logger_init", "status", "success")

	// core config
	coreCfg, err := coreConf.LoadCoreConfig("config/core.yml")
	if err != nil {
		log.Fatal(err)
	}
	b, err = yaml.Marshal(logCfg)
	if err != nil {
		log.Fatal(err)
	}
	logger.Info(
		"core_config_successfully_parsed",
		slog.String("config", string(b)),
	)

	// Database
	pool, err := postgres.NewPool(appCtx, coreCfg.Postgres)
	if err != nil {
		logger.Error("postgres_connection", "status", "error", "msg", err)
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("postgres_connection", "status", "success")

	queries := db.New(pool)

	// nats
	nc, err := nats.Connect(coreCfg.Nats.Url)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Drain()

	js, err := qnats.ConnectJetStream(nc)
	if err != nil {
		log.Fatalf("Error connecting stream: %v", err)
	}

	//logic
	publisher := qnats.NewPublisher(js)
	repository := repo.NewSchedulerRepository(pool, queries)
	schedulerService := service.NewSchedulerService(logger, repository, publisher)

	// semaphore
	sem := make(chan struct{}, 256)

	// Bacground checkers
	go schedulerService.MonitorHungedTasks(appCtx, coreCfg.BackgroundTasks.HungJobsMonitor.JobDeathSecondsTimeout,
		coreCfg.BackgroundTasks.HungJobsMonitor.PollInterval)

	go schedulerService.MonitorDisabledTasks(appCtx, coreCfg.BackgroundTasks.DisableJobsMonitor.PollInterval)

	// Loop
	ticker := time.NewTicker(coreCfg.GetJobPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-appCtx.Done():
			return
		case <-ticker.C:
			jID := schedulerService.ClaimNextJob(appCtx)
			if jID == uuid.Nil {
				continue
			}

			sem <- struct{}{}
			go func(id uuid.UUID) {
				defer func() { <-sem }()
				schedulerService.PublishJobIdToChannel(appCtx, id)
			}(jID)
		}
	}
}
