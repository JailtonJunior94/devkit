package tracing

import (
	"context"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// WithSampler sets the trace sampler used by the SDK provider.
func WithSampler(s sdktrace.Sampler) Option {
	return func(_ context.Context, cfg *Config) error {
		cfg.Sampler = s
		return nil
	}
}
