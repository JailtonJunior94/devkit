package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// WithSampler sets the trace sampler used by the SDK provider.
func WithSampler(s sdktrace.Sampler) Option {
	return func(_ context.Context, cfg *Config) error {
		cfg.Sampler = s
		return nil
	}
}

// WithOTLPGRPC configures an OTLP gRPC span exporter.
// The optional endpoint overrides the default
// (OTEL_EXPORTER_OTLP_ENDPOINT env var or localhost:4317).
func WithOTLPGRPC(endpoint ...string) Option {
	return func(ctx context.Context, cfg *Config) error {
		var opts []otlptracegrpc.Option
		if len(endpoint) > 0 && endpoint[0] != "" {
			opts = append(opts, otlptracegrpc.WithEndpoint(endpoint[0]))
		}
		exp, err := otlptracegrpc.New(ctx, opts...)
		if err != nil {
			return fmt.Errorf("tracing: creating OTLP gRPC exporter: %w", err)
		}
		cfg.SpanExporter = exp
		return nil
	}
}

// WithOTLPHTTP configures an OTLP HTTP span exporter.
// The optional endpoint overrides the default
// (OTEL_EXPORTER_OTLP_ENDPOINT env var or localhost:4318).
func WithOTLPHTTP(endpoint ...string) Option {
	return func(ctx context.Context, cfg *Config) error {
		var opts []otlptracehttp.Option
		if len(endpoint) > 0 && endpoint[0] != "" {
			opts = append(opts, otlptracehttp.WithEndpoint(endpoint[0]))
		}
		exp, err := otlptracehttp.New(ctx, opts...)
		if err != nil {
			return fmt.Errorf("tracing: creating OTLP HTTP exporter: %w", err)
		}
		cfg.SpanExporter = exp
		return nil
	}
}
