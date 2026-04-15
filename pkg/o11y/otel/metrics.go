package otel

import (
	"context"
	"fmt"

	observability "devkit/pkg/o11y"
	"go.opentelemetry.io/otel/metric"
)

type otelMetrics struct {
	meter metric.Meter
}

func newOtelMetrics(meter metric.Meter) *otelMetrics {
	return &otelMetrics{meter: meter}
}

func (m *otelMetrics) Counter(name, description, unit string) (observability.Counter, error) {
	counter, err := m.meter.Int64Counter(
		name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
	if err != nil {
		return nil, fmt.Errorf("creating counter %q: %w", name, err)
	}

	return &otelCounter{counter: counter}, nil
}

func (m *otelMetrics) Histogram(name, description, unit string) (observability.Histogram, error) {
	histogram, err := m.meter.Float64Histogram(
		name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
	if err != nil {
		return nil, fmt.Errorf("creating histogram %q: %w", name, err)
	}

	return &otelHistogram{histogram: histogram}, nil
}

func (m *otelMetrics) UpDownCounter(name, description, unit string) (observability.UpDownCounter, error) {
	upDown, err := m.meter.Int64UpDownCounter(
		name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
	if err != nil {
		return nil, fmt.Errorf("creating up-down counter %q: %w", name, err)
	}

	return &otelUpDownCounter{counter: upDown}, nil
}

func (m *otelMetrics) Gauge(name, description, unit string, callback observability.GaugeCallback) error {
	if callback == nil {
		return observability.ErrNilGaugeCallback
	}

	_, err := m.meter.Float64ObservableGauge(
		name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
		metric.WithFloat64Callback(func(ctx context.Context, observer metric.Float64Observer) error {
			ctx = normalizeContext(ctx)
			value := callback(ctx)
			observer.Observe(value)
			return nil
		}),
	)
	return err
}

type otelCounter struct {
	counter metric.Int64Counter
}

func (c *otelCounter) Add(ctx context.Context, value int64, fields ...observability.Field) {
	ctx = normalizeContext(ctx)
	attrs := convertFieldsToAttributes(fields)
	if attrs == nil {
		c.counter.Add(ctx, value)
		return
	}

	c.counter.Add(ctx, value, metric.WithAttributes(attrs...))
}

func (c *otelCounter) Increment(ctx context.Context, fields ...observability.Field) {
	c.Add(ctx, 1, fields...)
}

type otelHistogram struct {
	histogram metric.Float64Histogram
}

func (h *otelHistogram) Record(ctx context.Context, value float64, fields ...observability.Field) {
	ctx = normalizeContext(ctx)
	attrs := convertFieldsToAttributes(fields)
	if attrs == nil {
		h.histogram.Record(ctx, value)
		return
	}

	h.histogram.Record(ctx, value, metric.WithAttributes(attrs...))
}

type otelUpDownCounter struct {
	counter metric.Int64UpDownCounter
}

func (u *otelUpDownCounter) Add(ctx context.Context, value int64, fields ...observability.Field) {
	ctx = normalizeContext(ctx)
	attrs := convertFieldsToAttributes(fields)
	if attrs == nil {
		u.counter.Add(ctx, value)
		return
	}

	u.counter.Add(ctx, value, metric.WithAttributes(attrs...))
}

