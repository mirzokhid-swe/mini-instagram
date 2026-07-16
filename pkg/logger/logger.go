package logger

import (
	"log/slog"
	"os"
)

type Interface interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

type Logger struct {
	*slog.Logger
}

func New(level string) *Logger {
	var lvl slog.Level

	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelDebug
	}

	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: lvl,
	}))

	return &Logger{l}
}
