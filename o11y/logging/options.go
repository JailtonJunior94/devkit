package logging

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
)

// WithOTLPGRPC configures an OTLP gRPC log exporter.
// The optional endpoint overrides the default
// (OTEL_EXPORTER_OTLP_ENDPOINT env var or localhost:4317).
func WithOTLPGRPC(endpoint ...string) Option {
	return func(ctx context.Context, cfg *Config) error {
		var opts []otlploggrpc.Option
		if len(endpoint) > 0 && endpoint[0] != "" {
			opts = append(opts, otlploggrpc.WithEndpoint(endpoint[0]))
		}
		exp, err := otlploggrpc.New(ctx, opts...)
		if err != nil {
			return fmt.Errorf("logging: creating OTLP gRPC exporter: %w", err)
		}
		cfg.LogExporter = exp
		return nil
	}
}

// WithOTLPHTTP configures an OTLP HTTP log exporter.
// The optional endpoint overrides the default
// (OTEL_EXPORTER_OTLP_ENDPOINT env var or localhost:4318).
func WithOTLPHTTP(endpoint ...string) Option {
	return func(ctx context.Context, cfg *Config) error {
		var opts []otlploghttp.Option
		if len(endpoint) > 0 && endpoint[0] != "" {
			opts = append(opts, otlploghttp.WithEndpoint(endpoint[0]))
		}
		exp, err := otlploghttp.New(ctx, opts...)
		if err != nil {
			return fmt.Errorf("logging: creating OTLP HTTP exporter: %w", err)
		}
		cfg.LogExporter = exp
		return nil
	}
}
