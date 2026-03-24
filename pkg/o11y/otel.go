// Package o11y provides a unified bootstrap facade for OpenTelemetry
// tracing, metrics, and logging. It composes the individual signal providers
// without registering any global OTel state.
package o11y

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"devkit/pkg/o11y/logging"
	"devkit/pkg/o11y/metrics"
	"devkit/pkg/o11y/tracing"
)

// ErrServiceNameRequired is returned by New when Config.ServiceName is empty.
var ErrServiceNameRequired = errors.New("o11y: service name is required")

// Config holds the configuration for all three observability signals.
// Signal exporters may be set directly on this struct or via Option functions;
// options applied to New override the corresponding struct fields.
type Config struct {
	// ServiceName identifies the service. Required.
	ServiceName string
	// ServiceVersion is the semver or build tag of the service.
	ServiceVersion string
	// Environment is the deployment environment (e.g. "production", "staging").
	Environment string
	// ResourceAttributes are extra OTel resource attributes added to all signals.
	ResourceAttributes []attribute.KeyValue

	// TraceExporter sends completed spans. nil = noop TracerProvider.
	TraceExporter sdktrace.SpanExporter
	// TraceSampler controls which spans are recorded. nil = ParentBased(AlwaysSample).
	TraceSampler sdktrace.Sampler

	// MetricExporter sends metric data. nil = noop MeterProvider.
	MetricExporter sdkmetric.Exporter
	// MetricInterval controls the periodic collection interval. 0 = SDK default (60 s).
	MetricInterval time.Duration

	// LogExporter sends log records. nil = slog.Default() fallback.
	LogExporter sdklog.Exporter

	// Propagator, when non-nil, is registered globally via otel.SetTextMapPropagator.
	// Use WithW3CPropagators() to configure the W3C TraceContext + Baggage default.
	// Propagation is opt-in to avoid mutating global OTel state without explicit consent (RF10).
	Propagator propagation.TextMapPropagator
}

// Observability is the assembled observability provider holding all three signals.
type Observability struct {
	tracer *tracing.Provider
	meter  *metrics.Provider
	logger *logging.Provider
	once   sync.Once
}

// New creates a fully configured Observability from the given Config and options.
// Options override the corresponding Config fields.
// On partial failure, already-started providers are shut down before returning.
func New(ctx context.Context, cfg Config, opts ...Option) (*Observability, error) {
	if cfg.ServiceName == "" {
		return nil, ErrServiceNameRequired
	}

	// Apply options — each option may override exporter/sampler/interval fields.
	for _, opt := range opts {
		if err := opt(ctx, &cfg); err != nil {
			return nil, fmt.Errorf("o11y: applying option: %w", err)
		}
	}

	// Register propagator globally only when explicitly requested (RF10 opt-in).
	if cfg.Propagator != nil {
		otel.SetTextMapPropagator(cfg.Propagator)
	}

	tracingCfg := tracing.Config{
		ServiceName:        cfg.ServiceName,
		ServiceVersion:     cfg.ServiceVersion,
		Environment:        cfg.Environment,
		ResourceAttributes: cfg.ResourceAttributes,
		SpanExporter:       cfg.TraceExporter,
		Sampler:            cfg.TraceSampler,
	}
	tp, err := tracing.New(ctx, tracingCfg)
	if err != nil {
		return nil, fmt.Errorf("o11y: initializing tracing: %w", err)
	}

	metricsCfg := metrics.Config{
		ServiceName:        cfg.ServiceName,
		ServiceVersion:     cfg.ServiceVersion,
		Environment:        cfg.Environment,
		ResourceAttributes: cfg.ResourceAttributes,
		Exporter:           cfg.MetricExporter,
		Interval:           cfg.MetricInterval,
	}
	mp, err := metrics.New(ctx, metricsCfg)
	if err != nil {
		_ = tp.Shutdown(ctx)
		return nil, fmt.Errorf("o11y: initializing metrics: %w", err)
	}

	loggingCfg := logging.Config{
		ServiceName:        cfg.ServiceName,
		ServiceVersion:     cfg.ServiceVersion,
		Environment:        cfg.Environment,
		ResourceAttributes: cfg.ResourceAttributes,
		LogExporter:        cfg.LogExporter,
	}
	lp, err := logging.New(ctx, loggingCfg)
	if err != nil {
		_ = errors.Join(tp.Shutdown(ctx), mp.Shutdown(ctx))
		return nil, fmt.Errorf("o11y: initializing logging: %w", err)
	}

	return &Observability{
		tracer: tp,
		meter:  mp,
		logger: lp,
	}, nil
}

// TracerProvider returns the trace.TracerProvider for the service.
func (s *Observability) TracerProvider() trace.TracerProvider {
	return s.tracer.TracerProvider()
}

// MeterProvider returns the metric.MeterProvider for the service.
func (s *Observability) MeterProvider() metric.MeterProvider {
	return s.meter.MeterProvider()
}

// Logger returns the *slog.Logger for the service.
func (s *Observability) Logger() *slog.Logger {
	return s.logger.Logger()
}

// Shutdown shuts down all signal providers, collecting all errors.
// It is idempotent; subsequent calls return nil.
func (s *Observability) Shutdown(ctx context.Context) error {
	var err error
	s.once.Do(func() {
		err = errors.Join(
			s.tracer.Shutdown(ctx),
			s.meter.Shutdown(ctx),
			s.logger.Shutdown(ctx),
		)
	})
	return err
}
