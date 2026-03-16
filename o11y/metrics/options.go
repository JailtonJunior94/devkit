package metrics

import (
	"context"
	"time"
)

// WithInterval sets the periodic collection interval for the metric reader.
// A zero or negative value uses the SDK default (60 s).
func WithInterval(d time.Duration) Option {
	return func(_ context.Context, cfg *Config) error {
		cfg.Interval = d
		return nil
	}
}
