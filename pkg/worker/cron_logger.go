package worker

import (
	"log/slog"
)

type cronLogAdapter struct {
	logger *slog.Logger
}

func (a *cronLogAdapter) Info(msg string, keysAndValues ...any) {
	a.logger.Info(msg, keysAndValues...)
}

func (a *cronLogAdapter) Error(err error, msg string, keysAndValues ...any) {
	args := append([]any{"error", err}, keysAndValues...)
	a.logger.Error(msg, args...)
}
