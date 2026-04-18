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

func NewLogger(cfg *appconfig.LoggingConfig) (*slog.Logger, error) {
	writer, err := buildWriter(cfg)
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
	case "text":
		handler = slog.NewTextHandler(writer, opts)
	default:
		handler = slog.NewJSONHandler(writer, opts)
	}

	return slog.New(handler), nil
}

func buildWriter(cfg *appconfig.LoggingConfig) (io.Writer, error) {
	switch strings.ToLower(cfg.Output) {
	case "file":
		fileWriter, err := newRollingFileWriter(cfg.Logger)
		if err != nil {
			return nil, err
		}
		return fileWriter, nil
	default:
		return os.Stdout, nil
	}
}

func newRollingFileWriter(fileCfg *appconfig.FileSettings) (io.Writer, error) {
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
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelDebug, nil
	}
}
