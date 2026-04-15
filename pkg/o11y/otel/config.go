package otel

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	observability "devkit/pkg/o11y"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

// ErrConfigRequired is returned when a nil Config is passed to NewProvider.
var ErrConfigRequired = errors.New("otel: config cannot be nil")

// ErrServiceNameRequired is returned when Config.ServiceName is empty.
var ErrServiceNameRequired = errors.New("otel: service name is required")

// Config holds the OpenTelemetry provider configuration.
// Exporters are injected by the caller; when nil the corresponding signal
// falls back to a no-op provider so the bridge implementations stay usable.
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string

	// TraceExporter receives finished spans. When nil a no-op TracerProvider
	// is used and no spans leave the process.
	TraceExporter sdktrace.SpanExporter
	// TraceSampler controls which spans are recorded. Defaults to
	// ParentBased(AlwaysSample) when nil.
	TraceSampler sdktrace.Sampler

	// MetricExporter receives collected metrics. When nil a no-op
	// MeterProvider is used.
	MetricExporter sdkmetric.Exporter
	// MetricInterval sets the periodic reader interval. Zero uses the SDK
	// default.
	MetricInterval time.Duration

	// LogExporter receives log records via the OTel SDK pipeline. When nil
	// the logger writes to a local slog handler only.
	LogExporter sdklog.Exporter

	// LogLevel sets the minimum severity for the local slog handler.
	LogLevel observability.LogLevel
	// LogFormat selects text or JSON for the local slog handler.
	LogFormat observability.LogFormat

	// ResourceAttributes are merged into the OTel resource alongside the
	// service identity fields.
	ResourceAttributes map[string]string
}

// Provider adapts the simplified o11y API to the OpenTelemetry SDK.
type Provider struct {
	config         *Config
	tracerProvider trace.TracerProvider
	meterProvider  metric.MeterProvider
	loggerProvider *sdklog.LoggerProvider
	tracer         *otelTracer
	logger         *otelLogger
	metrics        *otelMetrics
	shutdownFuncs  []func(context.Context) error
	shutdownOnce   sync.Once
	shutdownErr    error
}

func validateConfig(config *Config) error {
	if config.ServiceName == "" {
		return ErrServiceNameRequired
	}

	return nil
}

// NewProvider initializes tracing, logging, and metrics with OpenTelemetry.
// Exporters that are nil cause the corresponding signal to use a no-op
// provider so callers always get a valid Provider back.
func NewProvider(ctx context.Context, config *Config) (*Provider, error) {
	if config == nil {
		return nil, ErrConfigRequired
	}

	if err := validateConfig(config); err != nil {
		return nil, err
	}

	provider := &Provider{
		config:        config,
		shutdownFuncs: make([]func(context.Context) error, 0),
	}

	res, err := provider.createResource(ctx)
	if err != nil {
		return nil, fmt.Errorf("otel: creating resource: %w", err)
	}

	provider.initTracerProvider(res)

	provider.initMeterProvider(res)

	provider.initLoggerProvider(res)

	provider.tracer = newOtelTracer(provider.tracerProvider.Tracer(config.ServiceName))
	provider.logger = newOtelLogger(
		config.LogLevel,
		config.LogFormat,
		config.ServiceName,
		provider.loggerProvider.Logger(config.ServiceName),
	)
	provider.metrics = newOtelMetrics(provider.meterProvider.Meter(config.ServiceName))

	return provider, nil
}

func (p *Provider) createResource(ctx context.Context) (*resource.Resource, error) {
	attrs := []resource.Option{
		resource.WithAttributes(
			semconv.ServiceName(p.config.ServiceName),
			semconv.ServiceVersion(p.config.ServiceVersion),
			semconv.DeploymentEnvironmentName(p.config.Environment),
		),
	}

	if len(p.config.ResourceAttributes) > 0 {
		customAttrs := make([]attribute.KeyValue, 0, len(p.config.ResourceAttributes))
		for k, v := range p.config.ResourceAttributes {
			customAttrs = append(customAttrs, attribute.String(k, v))
		}
		attrs = append(attrs, resource.WithAttributes(customAttrs...))
	}

	return resource.New(
		ctx,
		attrs...,
	)
}

func (p *Provider) initTracerProvider(res *resource.Resource) {
	if p.config.TraceExporter == nil {
		p.tracerProvider = tracenoop.NewTracerProvider()
		return
	}

	sampler := p.config.TraceSampler
	if sampler == nil {
		sampler = sdktrace.ParentBased(sdktrace.AlwaysSample())
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(p.config.TraceExporter),
	)

	p.tracerProvider = tp
	p.shutdownFuncs = append(p.shutdownFuncs, tp.Shutdown)
}

func (p *Provider) initMeterProvider(res *resource.Resource) {
	if p.config.MetricExporter == nil {
		p.meterProvider = metricnoop.NewMeterProvider()
		return
	}

	var readerOpts []sdkmetric.PeriodicReaderOption
	if p.config.MetricInterval > 0 {
		readerOpts = append(readerOpts, sdkmetric.WithInterval(p.config.MetricInterval))
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(p.config.MetricExporter, readerOpts...)),
	)

	p.meterProvider = mp
	p.shutdownFuncs = append(p.shutdownFuncs, mp.Shutdown)
}

func (p *Provider) initLoggerProvider(res *resource.Resource) {
	if p.config.LogExporter == nil {
		p.loggerProvider = sdklog.NewLoggerProvider()
		return
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(p.config.LogExporter)),
	)

	p.loggerProvider = lp
	p.shutdownFuncs = append(p.shutdownFuncs, lp.Shutdown)
}

// Tracer returns the simplified tracer backed by OpenTelemetry.
func (p *Provider) Tracer() observability.Tracer {
	return p.tracer
}

// Logger returns the simplified logger backed by OpenTelemetry.
func (p *Provider) Logger() observability.Logger {
	return p.logger
}

// Metrics returns the simplified metrics facade backed by OpenTelemetry.
func (p *Provider) Metrics() observability.Metrics {
	return p.metrics
}

// Shutdown closes the providers and attempts to flush pending telemetry.
func (p *Provider) Shutdown(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	p.shutdownOnce.Do(func() {
		var errs []error
		for i := len(p.shutdownFuncs) - 1; i >= 0; i-- {
			if err := p.shutdownFuncs[i](ctx); err != nil {
				errs = append(errs, err)
			}
		}

		if len(errs) > 0 {
			p.shutdownErr = fmt.Errorf("shutdown provider: %w", errors.Join(errs...))
		}
	})

	return p.shutdownErr
}

// DefaultConfig returns a baseline configuration for a service.
func DefaultConfig(serviceName string) *Config {
	return &Config{
		ServiceName:    serviceName,
		ServiceVersion: "unknown",
		Environment:    "development",
		LogLevel:       observability.LogLevelInfo,
		LogFormat:      observability.LogFormatJSON,
	}
}
