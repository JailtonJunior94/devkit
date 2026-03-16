package o11y

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Option configures the Observability bootstrap.
// Options are applied after Config fields, allowing them to override any
// exporter, sampler, or interval already set in Config.
type Option func(ctx context.Context, cfg *Config) error

// WithSampler sets the trace sampler for the SDK.
func WithSampler(s sdktrace.Sampler) Option {
	return func(_ context.Context, cfg *Config) error {
		cfg.TraceSampler = s
		return nil
	}
}

// WithSpanExporter sets the span exporter directly, useful for testing
// or when a custom exporter is provided by the caller.
func WithSpanExporter(exp sdktrace.SpanExporter) Option {
	return func(_ context.Context, cfg *Config) error {
		cfg.TraceExporter = exp
		return nil
	}
}

// WithMetricExporter sets the metric exporter directly, useful for testing
// or when a custom exporter is provided by the caller.
func WithMetricExporter(exp sdkmetric.Exporter) Option {
	return func(_ context.Context, cfg *Config) error {
		cfg.MetricExporter = exp
		return nil
	}
}

// WithMetricInterval sets the periodic collection interval for the metric reader.
// A zero or negative value uses the SDK default (60 s).
func WithMetricInterval(d time.Duration) Option {
	return func(_ context.Context, cfg *Config) error {
		cfg.MetricInterval = d
		return nil
	}
}

// WithLogExporter sets the log exporter directly, useful for testing
// or when a custom exporter is provided by the caller.
func WithLogExporter(exp sdklog.Exporter) Option {
	return func(_ context.Context, cfg *Config) error {
		cfg.LogExporter = exp
		return nil
	}
}

// WithW3CPropagators registers the W3C TraceContext and Baggage propagators
// globally via otel.SetTextMapPropagator. This is the opt-in mechanism for
// distributed trace context propagation across service boundaries (RF01/RF10).
//
// Without this option, trace context is not propagated via HTTP headers and
// spans from different services will not be connected automatically.
func WithW3CPropagators() Option {
	return func(_ context.Context, cfg *Config) error {
		cfg.Propagator = propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		)
		return nil
	}
}
