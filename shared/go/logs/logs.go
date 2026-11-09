package logs

import (
	"log/slog"
	"os"
)

func New(service string) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level()})
	return slog.New(handler).With("service", service)
}

func level() slog.Level {
	return slog.LevelInfo
}
