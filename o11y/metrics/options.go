package metrics

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
)

// WithInterval sets the periodic collection interval for the metric reader.
// A zero or negative value uses the SDK default (60 s).
func WithInterval(d time.Duration) Option {
	return func(_ context.Context, cfg *Config) error {
		cfg.Interval = d
		return nil
	}
}

// WithOTLPGRPC configures an OTLP gRPC metric exporter.
// The optional endpoint overrides the default
// (OTEL_EXPORTER_OTLP_ENDPOINT env var or localhost:4317).
func WithOTLPGRPC(endpoint ...string) Option {
	return func(ctx context.Context, cfg *Config) error {
		var opts []otlpmetricgrpc.Option
		if len(endpoint) > 0 && endpoint[0] != "" {
			opts = append(opts, otlpmetricgrpc.WithEndpoint(endpoint[0]))
		}
		exp, err := otlpmetricgrpc.New(ctx, opts...)
		if err != nil {
			return fmt.Errorf("metrics: creating OTLP gRPC exporter: %w", err)
		}
		cfg.Exporter = exp
		return nil
	}
}

// WithOTLPHTTP configures an OTLP HTTP metric exporter.
// The optional endpoint overrides the default
// (OTEL_EXPORTER_OTLP_ENDPOINT env var or localhost:4318).
func WithOTLPHTTP(endpoint ...string) Option {
	return func(ctx context.Context, cfg *Config) error {
		var opts []otlpmetrichttp.Option
		if len(endpoint) > 0 && endpoint[0] != "" {
			opts = append(opts, otlpmetrichttp.WithEndpoint(endpoint[0]))
		}
		exp, err := otlpmetrichttp.New(ctx, opts...)
		if err != nil {
			return fmt.Errorf("metrics: creating OTLP HTTP exporter: %w", err)
		}
		cfg.Exporter = exp
		return nil
	}
}
