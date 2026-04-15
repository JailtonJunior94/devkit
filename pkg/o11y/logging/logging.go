// Package logging provides bootstrap helpers for OpenTelemetry-backed structured logging.
package logging

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/attribute"
	sdklog "go.opentelemetry.io/otel/sdk/log"

	"devkit/pkg/o11y/internal/resource"
)

// ErrServiceNameRequired is returned when ServiceName is empty.
var ErrServiceNameRequired = errors.New("logging: service name is required")

// ErrNilOption is returned when a nil Option is passed to New.
var ErrNilOption = errors.New("logging: option cannot be nil")

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
	// When nil, the provider uses an isolated discard-backed logger and no OTel pipeline is created.
	LogExporter sdklog.Exporter
	// Handler is an optional slog.Handler composed with the OTel bridge when
	// LogExporter is configured. When LogExporter is nil, Handler becomes the
	// active logger backend directly.
	Handler slog.Handler
}

// Option configures logging bootstrap.
type Option func(ctx context.Context, cfg *Config) error

// Provider wraps a configured slog.Logger backed by an OTel LoggerProvider.
type Provider struct {
	logger   *slog.Logger
	shutdown func(context.Context) error
	once     sync.Once
}

// New creates a configured logging provider.
func New(ctx context.Context, cfg Config, opts ...Option) (*Provider, error) {
	if cfg.ServiceName == "" {
		return nil, ErrServiceNameRequired
	}
	for _, opt := range opts {
		if opt == nil {
			return nil, ErrNilOption
		}
		if err := opt(ctx, &cfg); err != nil {
			return nil, fmt.Errorf("logging: applying option: %w", err)
		}
	}
	if cfg.LogExporter == nil {
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		if cfg.Handler != nil {
			logger = slog.New(cfg.Handler)
		}
		return &Provider{
			logger:   logger,
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
	handler := slog.Handler(otelslog.NewHandler(cfg.ServiceName, otelslog.WithLoggerProvider(lp)))
	if cfg.Handler != nil {
		handler = newMultiHandler(handler, cfg.Handler)
	}
	return &Provider{
		logger:   slog.New(handler),
		shutdown: lp.Shutdown,
	}, nil
}

// Logger returns the slog logger for the service.
func (p *Provider) Logger() *slog.Logger {
	return p.logger
}

// Shutdown flushes pending log records and releases resources.
func (p *Provider) Shutdown(ctx context.Context) error {
	var err error
	p.once.Do(func() {
		err = p.shutdown(ctx)
	})
	return err
}

type multiHandler struct {
	handlers []slog.Handler
}

func newMultiHandler(handlers ...slog.Handler) slog.Handler {
	filtered := make([]slog.Handler, 0, len(handlers))
	for _, handler := range handlers {
		if handler != nil {
			filtered = append(filtered, handler)
		}
	}
	if len(filtered) == 1 {
		return filtered[0]
	}
	return &multiHandler{handlers: filtered}
}

func (h *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *multiHandler) Handle(ctx context.Context, record slog.Record) error {
	var err error
	for _, handler := range h.handlers {
		if !handler.Enabled(ctx, record.Level) {
			continue
		}
		err = errors.Join(err, handler.Handle(ctx, record.Clone()))
	}
	return err
}

func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := make([]slog.Handler, 0, len(h.handlers))
	for _, handler := range h.handlers {
		next = append(next, handler.WithAttrs(attrs))
	}
	return &multiHandler{handlers: next}
}

func (h *multiHandler) WithGroup(name string) slog.Handler {
	next := make([]slog.Handler, 0, len(h.handlers))
	for _, handler := range h.handlers {
		next = append(next, handler.WithGroup(name))
	}
	return &multiHandler{handlers: next}
}
