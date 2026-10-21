package main

import (
	"context"
	"log/slog"
	"strings"
)

const logSource = "frontend"

// LogService receives browser-side logs through an explicit Wails
// service. It deliberately does not expose Fatal: frontend code must never be
// able to terminate the desktop process.
type LogService struct {
	logger *slog.Logger
}

func NewLogService(logger *slog.Logger) *LogService {
	if logger == nil {
		logger = slog.Default()
	}

	return &LogService{
		logger: logger,
	}
}

func (s *LogService) Debug(message string) {
	s.log(slog.LevelDebug, message)
}

func (s *LogService) Info(message string) {
	s.log(slog.LevelInfo, message)
}

func (s *LogService) Warn(message string) {
	s.log(slog.LevelWarn, message)
}

func (s *LogService) Error(message string) {
	s.log(slog.LevelError, message)
}

func (s *LogService) log(level slog.Level, message string) {
	if strings.TrimSpace(message) == "" {
		return
	}

	logger := slog.Default()
	if s != nil && s.logger != nil {
		logger = s.logger
	}

	logger.Log(
		context.Background(),
		level,
		message,
		"source",
		logSource,
	)
}
