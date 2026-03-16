package o11y

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
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

// WithOTLPTraceGRPC configures an OTLP gRPC span exporter.
// The optional endpoint overrides the default
// (OTEL_EXPORTER_OTLP_ENDPOINT env var or localhost:4317).
func WithOTLPTraceGRPC(endpoint ...string) Option {
	return func(ctx context.Context, cfg *Config) error {
		var opts []otlptracegrpc.Option
		if len(endpoint) > 0 && endpoint[0] != "" {
			opts = append(opts, otlptracegrpc.WithEndpoint(endpoint[0]))
		}
		exp, err := otlptracegrpc.New(ctx, opts...)
		if err != nil {
			return fmt.Errorf("o11y: creating OTLP gRPC trace exporter: %w", err)
		}
		cfg.TraceExporter = exp
		return nil
	}
}

// WithOTLPTraceHTTP configures an OTLP HTTP span exporter.
// The optional endpoint overrides the default
// (OTEL_EXPORTER_OTLP_ENDPOINT env var or localhost:4318).
func WithOTLPTraceHTTP(endpoint ...string) Option {
	return func(ctx context.Context, cfg *Config) error {
		var opts []otlptracehttp.Option
		if len(endpoint) > 0 && endpoint[0] != "" {
			opts = append(opts, otlptracehttp.WithEndpoint(endpoint[0]))
		}
		exp, err := otlptracehttp.New(ctx, opts...)
		if err != nil {
			return fmt.Errorf("o11y: creating OTLP HTTP trace exporter: %w", err)
		}
		cfg.TraceExporter = exp
		return nil
	}
}

// WithOTLPMetricGRPC configures an OTLP gRPC metric exporter.
// The optional endpoint overrides the default
// (OTEL_EXPORTER_OTLP_ENDPOINT env var or localhost:4317).
func WithOTLPMetricGRPC(endpoint ...string) Option {
	return func(ctx context.Context, cfg *Config) error {
		var opts []otlpmetricgrpc.Option
		if len(endpoint) > 0 && endpoint[0] != "" {
			opts = append(opts, otlpmetricgrpc.WithEndpoint(endpoint[0]))
		}
		exp, err := otlpmetricgrpc.New(ctx, opts...)
		if err != nil {
			return fmt.Errorf("o11y: creating OTLP gRPC metric exporter: %w", err)
		}
		cfg.MetricExporter = exp
		return nil
	}
}

// WithOTLPMetricHTTP configures an OTLP HTTP metric exporter.
// The optional endpoint overrides the default
// (OTEL_EXPORTER_OTLP_ENDPOINT env var or localhost:4318).
func WithOTLPMetricHTTP(endpoint ...string) Option {
	return func(ctx context.Context, cfg *Config) error {
		var opts []otlpmetrichttp.Option
		if len(endpoint) > 0 && endpoint[0] != "" {
			opts = append(opts, otlpmetrichttp.WithEndpoint(endpoint[0]))
		}
		exp, err := otlpmetrichttp.New(ctx, opts...)
		if err != nil {
			return fmt.Errorf("o11y: creating OTLP HTTP metric exporter: %w", err)
		}
		cfg.MetricExporter = exp
		return nil
	}
}

// WithOTLPLogGRPC configures an OTLP gRPC log exporter.
// The optional endpoint overrides the default
// (OTEL_EXPORTER_OTLP_ENDPOINT env var or localhost:4317).
func WithOTLPLogGRPC(endpoint ...string) Option {
	return func(ctx context.Context, cfg *Config) error {
		var opts []otlploggrpc.Option
		if len(endpoint) > 0 && endpoint[0] != "" {
			opts = append(opts, otlploggrpc.WithEndpoint(endpoint[0]))
		}
		exp, err := otlploggrpc.New(ctx, opts...)
		if err != nil {
			return fmt.Errorf("o11y: creating OTLP gRPC log exporter: %w", err)
		}
		cfg.LogExporter = exp
		return nil
	}
}

// WithOTLPLogHTTP configures an OTLP HTTP log exporter.
// The optional endpoint overrides the default
// (OTEL_EXPORTER_OTLP_ENDPOINT env var or localhost:4318).
func WithOTLPLogHTTP(endpoint ...string) Option {
	return func(ctx context.Context, cfg *Config) error {
		var opts []otlploghttp.Option
		if len(endpoint) > 0 && endpoint[0] != "" {
			opts = append(opts, otlploghttp.WithEndpoint(endpoint[0]))
		}
		exp, err := otlploghttp.New(ctx, opts...)
		if err != nil {
			return fmt.Errorf("o11y: creating OTLP HTTP log exporter: %w", err)
		}
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
