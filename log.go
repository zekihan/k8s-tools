package main

import (
	"github.com/lmittmann/tint"
	"log/slog"
	"os"
	"time"
)

func initLogger(s string) {
	level := slog.LevelInfo

	switch s {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	case "":
		level = slog.LevelInfo
	default:
		slog.Warn("Invalid log level, using info", slog.String("level", s))
	}

	var handler slog.Handler
	if info, err := os.Stdout.Stat(); err == nil && info.Mode()&os.ModeCharDevice != 0 {
		handler = tint.NewTextHandler(os.Stdout, &tint.Options{
			Level:      level,
			TimeFormat: time.RFC3339,
		})
	} else {
		// Preserve literal diagnostic text in pipes and files.
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	}
	slog.SetDefault(slog.New(handler))
}
