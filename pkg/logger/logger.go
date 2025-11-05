package logger

import (
	"log/slog"
	"os"
	"strings"
)

// InitLogger инициализирует глобальный логгер с указанным уровнем
func InitLogger(levelStr string) {
	var logLevel slog.Level
	
	switch strings.ToLower(levelStr) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn", "warning":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)
}