package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	appconfig "github.com/flamefks/scheduler-system/internal/config"
	"gopkg.in/natefinch/lumberjack.v2"
)

func New(cfg *appconfig.LoggingConfig, fileCfg appconfig.FileSettings) (*slog.Logger, error) {
	writer, err := buildWriter(cfg, fileCfg)
	if err != nil {
		return nil, err
	}

	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler

	switch strings.ToLower(cfg.Format) {
	case "json":
		handler = slog.NewJSONHandler(writer, opts)
	case "text":
		handler = slog.NewTextHandler(writer, opts)
	default:
		return nil, fmt.Errorf("unsupported log format: %s", cfg.Format)
	}

	return slog.New(handler), nil
}

func buildWriter(cfg *appconfig.LoggingConfig, fileCfg appconfig.FileSettings) (io.Writer, error) {
	switch strings.ToLower(cfg.Output) {
	case "stdout":
		return os.Stdout, nil

	case "file":
		fileWriter, err := newRollingFileWriter(fileCfg)
		if err != nil {
			return nil, err
		}
		return fileWriter, nil

	case "both":
		fileWriter, err := newRollingFileWriter(fileCfg)
		if err != nil {
			return nil, err
		}
		return io.MultiWriter(os.Stdout, fileWriter), nil

	default:
		return nil, fmt.Errorf("unsupported log output: %s", cfg.Output)
	}
}

func newRollingFileWriter(fileCfg appconfig.FileSettings) (io.Writer, error) {
	if fileCfg.Path == "" {
		return nil, fmt.Errorf("logging file path is required")
	}

	dir := filepath.Dir(fileCfg.Path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create log dir: %w", err)
		}
	}

	return &lumberjack.Logger{
		Filename:   fileCfg.Path,
		MaxSize:    fileCfg.MaxSizeMB,
		MaxBackups: fileCfg.MaxBackups,
		MaxAge:     fileCfg.MaxAgeDays,
		Compress:   fileCfg.Compress,
	}, nil
}

func parseLevel(raw string) (slog.Level, error) {
	switch strings.ToLower(raw) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unsupported log level: %s", raw)
	}
}
