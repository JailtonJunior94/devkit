// Package metrics provides bootstrap helpers for OpenTelemetry metrics.
package metrics

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"devkit/pkg/o11y/internal/resource"
)

// ErrServiceNameRequired is returned when ServiceName is empty.
var ErrServiceNameRequired = errors.New("metrics: service name is required")

// ErrNilOption is returned when a nil Option is passed to New.
var ErrNilOption = errors.New("metrics: option cannot be nil")

// Config holds metrics-specific configuration.
type Config struct {
	ServiceName        string
	ServiceVersion     string
	Environment        string
	ResourceAttributes []attribute.KeyValue
	Exporter           sdkmetric.Exporter
	Interval           time.Duration
}

// Option configures metrics bootstrap.
type Option func(ctx context.Context, cfg *Config) error

// Provider wraps a configured MeterProvider with managed shutdown.
type Provider struct {
	provider metric.MeterProvider
	shutdown func(context.Context) error
	once     sync.Once
}

// New creates a configured MeterProvider.
func New(ctx context.Context, cfg Config, opts ...Option) (*Provider, error) {
	if cfg.ServiceName == "" {
		return nil, ErrServiceNameRequired
	}
	for _, opt := range opts {
		if opt == nil {
			return nil, ErrNilOption
		}
		if err := opt(ctx, &cfg); err != nil {
			return nil, fmt.Errorf("metrics: applying option: %w", err)
		}
	}
	if cfg.Exporter == nil {
		return &Provider{
			provider: metricnoop.NewMeterProvider(),
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
		return nil, fmt.Errorf("metrics: building resource: %w", err)
	}

	var readerOpts []sdkmetric.PeriodicReaderOption
	if cfg.Interval > 0 {
		readerOpts = append(readerOpts, sdkmetric.WithInterval(cfg.Interval))
	}
	reader := sdkmetric.NewPeriodicReader(cfg.Exporter, readerOpts...)
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
		sdkmetric.WithResource(res),
	)
	return &Provider{
		provider: mp,
		shutdown: mp.Shutdown,
	}, nil
}

// MeterProvider returns the underlying metric provider.
func (p *Provider) MeterProvider() metric.MeterProvider {
	return p.provider
}

// Shutdown flushes pending metrics and releases resources.
func (p *Provider) Shutdown(ctx context.Context) error {
	var err error
	p.once.Do(func() {
		err = p.shutdown(ctx)
	})
	return err
}
