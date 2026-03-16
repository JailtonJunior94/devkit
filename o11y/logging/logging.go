// Package logging provides bootstrap helpers for OpenTelemetry-backed structured logging.
package logging

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/attribute"
	sdklog "go.opentelemetry.io/otel/sdk/log"

	"devkit/o11y/internal/resource"
)

// ErrServiceNameRequired is returned when ServiceName is empty.
var ErrServiceNameRequired = errors.New("logging: service name is required")

// Config holds logging-specific configuration.
type Config struct {
	// ServiceName is the logical name of the service. Required.
	ServiceName string
	// ServiceVersion is the version string of the service (e.g. "1.2.3"). Optional.
	ServiceVersion string
	// Environment identifies the deployment environment (e.g. "prod", "staging"). Optional.
	Environment string
	// ResourceAttributes are additional OTel resource attributes merged into the resource. Optional.
	ResourceAttributes []attribute.KeyValue
	// LogExporter is the OTel log exporter.
	// When nil, the provider falls back to slog.Default() and no OTel pipeline is created.
	LogExporter sdklog.Exporter
}

// Option configures logging bootstrap.
type Option func(ctx context.Context, cfg *Config) error

// Provider wraps a configured slog.Logger backed by an OTel LoggerProvider.
type Provider struct {
	logger   *slog.Logger
	shutdown func(context.Context) error
	once     sync.Once
}

// New creates a configured logging Provider.
// When no LogExporter is configured, it falls back to slog.Default().
func New(ctx context.Context, cfg Config, opts ...Option) (*Provider, error) {
	if cfg.ServiceName == "" {
		return nil, ErrServiceNameRequired
	}
	for _, opt := range opts {
		if err := opt(ctx, &cfg); err != nil {
			return nil, fmt.Errorf("logging: applying option: %w", err)
		}
	}
	if cfg.LogExporter == nil {
		return &Provider{
			logger:   slog.Default(),
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
		return nil, fmt.Errorf("logging: building resource: %w", err)
	}
	processor := sdklog.NewBatchProcessor(cfg.LogExporter)
	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(processor),
		sdklog.WithResource(res),
	)
	handler := otelslog.NewHandler(cfg.ServiceName, otelslog.WithLoggerProvider(lp))
	logger := slog.New(handler)
	return &Provider{
		logger:   logger,
		shutdown: lp.Shutdown,
	}, nil
}

// Logger returns the *slog.Logger for the service.
func (p *Provider) Logger() *slog.Logger {
	return p.logger
}

// Shutdown flushes pending log records and releases resources. It is idempotent;
// subsequent calls return nil.
func (p *Provider) Shutdown(ctx context.Context) error {
	var err error
	p.once.Do(func() {
		err = p.shutdown(ctx)
	})
	return err
}
