package logging

import (
	"context"
	"log/slog"
)

// WithHandler configures a custom slog.Handler for the logger. When a log
// exporter is configured, the handler is composed with the OTel bridge so the
// record is written to both backends.
func WithHandler(handler slog.Handler) Option {
	return func(_ context.Context, cfg *Config) error {
		cfg.Handler = handler
		return nil
	}
}
