package noop

import (
	"context"

	observability "devkit/pkg/o11y"
)

type noopSpanContextKey struct{}

// Provider disables observability while keeping the same application-facing API.
type Provider struct {
	tracer  *noopTracer
	logger  *noopLogger
	metrics *noopMetrics
}

// NewProvider creates a no-op implementation of the simplified o11y contracts.
func NewProvider() *Provider {
	return &Provider{
		tracer:  &noopTracer{},
		logger:  &noopLogger{},
		metrics: &noopMetrics{},
	}
}

// Tracer returns the no-op tracer implementation.
func (p *Provider) Tracer() observability.Tracer {
	return p.tracer
}

// Logger returns the no-op logger implementation.
func (p *Provider) Logger() observability.Logger {
	return p.logger
}

// Metrics returns the no-op metrics implementation.
func (p *Provider) Metrics() observability.Metrics {
	return p.metrics
}

type noopTracer struct{}

func (t *noopTracer) Start(ctx context.Context, spanName string, opts ...observability.SpanOption) (context.Context, observability.Span) {
	ctx = ensureNoopContext(ctx)
	span := noopSpan{}
	return context.WithValue(ctx, noopSpanContextKey{}, span), span
}

func (t *noopTracer) SpanFromContext(ctx context.Context) observability.Span {
	ctx = ensureNoopContext(ctx)
	if span, ok := ctx.Value(noopSpanContextKey{}).(observability.Span); ok && span != nil {
		return span
	}

	return noopSpan{}
}

func (t *noopTracer) ContextWithSpan(ctx context.Context, span observability.Span) context.Context {
	ctx = ensureNoopContext(ctx)
	if span == nil {
		return ctx
	}

	return context.WithValue(ctx, noopSpanContextKey{}, span)
}

type noopSpan struct{}

func (s noopSpan) End() {}

func (s noopSpan) SetAttributes(fields ...observability.Field) {}

func (s noopSpan) SetStatus(code observability.StatusCode, description string) {}

func (s noopSpan) RecordError(err error, fields ...observability.Field) {}

func (s noopSpan) AddEvent(name string, fields ...observability.Field) {}

func (s noopSpan) Context() observability.SpanContext {
	return noopSpanContext{}
}

type noopSpanContext struct{}

func (c noopSpanContext) TraceID() string {
	return ""
}

func (c noopSpanContext) SpanID() string {
	return ""
}

func (c noopSpanContext) IsSampled() bool {
	return false
}

type noopLogger struct{}

func (l *noopLogger) Debug(ctx context.Context, msg string, fields ...observability.Field) {}

func (l *noopLogger) Info(ctx context.Context, msg string, fields ...observability.Field) {}

func (l *noopLogger) Warn(ctx context.Context, msg string, fields ...observability.Field) {}

func (l *noopLogger) Error(ctx context.Context, msg string, fields ...observability.Field) {}

func (l *noopLogger) With(fields ...observability.Field) observability.Logger {
	return l
}

type noopMetrics struct{}

func (m *noopMetrics) Counter(name, description, unit string) (observability.Counter, error) {
	return noopCounter{}, nil
}

func (m *noopMetrics) Histogram(name, description, unit string) (observability.Histogram, error) {
	return noopHistogram{}, nil
}

func (m *noopMetrics) UpDownCounter(name, description, unit string) (observability.UpDownCounter, error) {
	return noopUpDownCounter{}, nil
}

func (m *noopMetrics) Gauge(name, description, unit string, callback observability.GaugeCallback) error {
	if callback == nil {
		return observability.ErrNilGaugeCallback
	}

	return nil
}

type noopCounter struct{}

func (c noopCounter) Add(ctx context.Context, value int64, fields ...observability.Field) {}

func (c noopCounter) Increment(ctx context.Context, fields ...observability.Field) {}

type noopHistogram struct{}

func (h noopHistogram) Record(ctx context.Context, value float64, fields ...observability.Field) {}

type noopUpDownCounter struct{}

func (u noopUpDownCounter) Add(ctx context.Context, value int64, fields ...observability.Field) {}

func ensureNoopContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}

	return ctx
}
