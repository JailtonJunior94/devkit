package logging

import (
	"context"
	"log/slog"
)

// WithHandler configures a custom slog handler for the logger.
func WithHandler(handler slog.Handler) Option {
	return func(_ context.Context, cfg *Config) error {
		cfg.Handler = handler
		return nil
	}
}
