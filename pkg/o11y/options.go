package o11y

import (
	"context"
	"log/slog"
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

// WithHandler sets a custom slog.Handler. When a log exporter is configured,
// the handler is composed with the OTel slog bridge.
func WithHandler(handler slog.Handler) Option {
	return func(_ context.Context, cfg *Config) error {
		cfg.Handler = handler
		return nil
	}
}

// WithPropagator overrides the facade propagator without mutating global OTel state.
func WithPropagator(prop propagation.TextMapPropagator) Option {
	return func(_ context.Context, cfg *Config) error {
		cfg.Propagator = prop
		return nil
	}
}

// WithW3CPropagators keeps the default W3C TraceContext + Baggage propagator
// and registers it globally via otel.SetTextMapPropagator as an opt-in.
func WithW3CPropagators() Option {
	return func(_ context.Context, cfg *Config) error {
		cfg.Propagator = propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		)
		cfg.RegisterPropagatorGlobal = true
		return nil
	}
}
