package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	coreConf "github.com/flamefks/scheduler-system/internal/api/config"
	service "github.com/flamefks/scheduler-system/internal/api/service"
	apiHttp "github.com/flamefks/scheduler-system/internal/api/transport/http"

	dbRepo "github.com/flamefks/scheduler-system/internal/api/repository"
	generalConf "github.com/flamefks/scheduler-system/internal/config"
	logging "github.com/flamefks/scheduler-system/internal/logger"
	"github.com/flamefks/scheduler-system/internal/postgres"
	db "github.com/flamefks/scheduler-system/internal/postgres/queries"
)

func main() {
	ctx := context.Background()

	logCfg, err := generalConf.LoadLogging("config/logging.yml")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Logging config successfully parsed: %v", logCfg)

	coreCfg, err := coreConf.LoadCoreConfig("config/core.yml")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Core config successfully parsed: %v", coreCfg)

	logger, err := logging.NewLogger(logCfg)
	if err != nil {
		log.Fatal(err)
	}
	logger.Info(
		"logger_init",
		slog.String("status", "success"),
	)

	pool, err := postgres.NewPool(ctx, coreCfg.Postgres)
	if err != nil {
		logger.Error(
			"postgres_connection",
			slog.String("status", "error"),
			slog.Any("msg", err),
		)
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info(
		"postgres_connection",
		slog.String("status", "success"),
	)

	queries := db.New(pool)

	repo := dbRepo.NewRepository(pool, queries)
	apiService := service.NewApiService(logger, repo)
	apiHandler := apiHttp.NewApiHandler(apiService)
	router := apiHttp.NewRouter(apiHandler)

	srv := &http.Server{
		Addr:         coreCfg.HTTP.Host + ":" + strconv.Itoa(coreCfg.HTTP.Port),
		Handler:      router,
		ReadTimeout:  coreCfg.HTTP.ReadTimeout,
		WriteTimeout: coreCfg.HTTP.WriteTimeout,
		IdleTimeout:  coreCfg.HTTP.IdleTimeout,
	}

	logger.Info(
		"http server starting",
		slog.String("addr", srv.Addr),
	)

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("http server failed", "err", err)
		os.Exit(1)
	}
}
