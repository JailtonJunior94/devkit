// Package tracing provides bootstrap helpers for OpenTelemetry tracing.
package tracing

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	"devkit/o11y/internal/resource"
)

// ErrServiceNameRequired is returned when ServiceName is empty.
var ErrServiceNameRequired = errors.New("tracing: service name is required")

// Config holds tracing-specific configuration.
type Config struct {
	// ServiceName is the logical name of the service. Required.
	ServiceName string
	// ServiceVersion is the version string of the service (e.g. "1.2.3"). Optional.
	ServiceVersion string
	// Environment identifies the deployment environment (e.g. "prod", "staging"). Optional.
	Environment string
	// ResourceAttributes are additional OTel resource attributes merged into the resource. Optional.
	ResourceAttributes []attribute.KeyValue
	// SpanExporter is the exporter for completed spans.
	// When nil, a no-op TracerProvider is used and all spans are discarded.
	SpanExporter sdktrace.SpanExporter
	// Sampler overrides the default sampler (ParentBased(AlwaysSample)). Optional.
	Sampler sdktrace.Sampler
}

// Option configures tracing bootstrap.
type Option func(ctx context.Context, cfg *Config) error

// Provider wraps a configured TracerProvider with managed shutdown.
type Provider struct {
	provider trace.TracerProvider
	shutdown func(context.Context) error
	once     sync.Once
}

// New creates a configured TracerProvider.
// When no SpanExporter is configured (either via cfg or opts), it returns a
// no-op provider that discards all spans.
func New(ctx context.Context, cfg Config, opts ...Option) (*Provider, error) {
	if cfg.ServiceName == "" {
		return nil, ErrServiceNameRequired
	}
	for _, opt := range opts {
		if err := opt(ctx, &cfg); err != nil {
			return nil, fmt.Errorf("tracing: applying option: %w", err)
		}
	}
	if cfg.SpanExporter == nil {
		return &Provider{
			provider: tracenoop.NewTracerProvider(),
			shutdown: func(context.Context) error { return nil },
		}, nil
	}
	res, err := resource.Build(resource.Config{
		ServiceName:    cfg.ServiceName,
		ServiceVersion: cfg.ServiceVersion,
		Environment:    cfg.Environment,
		Attributes:     cfg.ResourceAttributes,
	})
	if err != nil {
		return nil, fmt.Errorf("tracing: building resource: %w", err)
	}
	sampler := cfg.Sampler
	if sampler == nil {
		sampler = sdktrace.ParentBased(sdktrace.AlwaysSample())
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(cfg.SpanExporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)
	return &Provider{
		provider: tp,
		shutdown: tp.Shutdown,
	}, nil
}

// TracerProvider returns the underlying trace.TracerProvider.
func (p *Provider) TracerProvider() trace.TracerProvider {
	return p.provider
}

// Shutdown flushes pending spans and releases resources. It is idempotent;
// subsequent calls return nil.
func (p *Provider) Shutdown(ctx context.Context) error {
	var err error
	p.once.Do(func() {
		err = p.shutdown(ctx)
	})
	return err
}
