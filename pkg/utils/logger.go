package utils

import (
	"log/slog"
	"os"
)

func InitLogger(verbose, silent bool) {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	} else if silent {
		level = slog.LevelError + 1
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)
}
