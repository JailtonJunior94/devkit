// Package noop exposes explicit zero-cost observability primitives for callers
// that want to disable telemetry without relying on implicit fallbacks.
package noop

import (
	"log/slog"

	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

// NewTracerProvider returns a trace.TracerProvider that discards all spans.
func NewTracerProvider() trace.TracerProvider {
	return tracenoop.NewTracerProvider()
}

// NewMeterProvider returns a metric.MeterProvider that discards all metrics.
func NewMeterProvider() metric.MeterProvider {
	return metricnoop.NewMeterProvider()
}

// NewLogger returns an slog.Logger that discards all log records.
func NewLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
