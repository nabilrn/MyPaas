package logger

import (
	"log/slog"
	"os"
)

func Setup(levelName string) {
	level := slog.LevelInfo
	if levelName == "debug" {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})))
}
