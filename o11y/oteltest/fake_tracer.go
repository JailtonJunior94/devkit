// Package oteltest provides in-memory test doubles for OpenTelemetry signals.
package oteltest

import (
	"context"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// FakeTracer provides an in-memory TracerProvider for test span inspection.
// Spans are exported synchronously and available immediately after span.End().
type FakeTracer struct {
	exporter *tracetest.InMemoryExporter
	provider *sdktrace.TracerProvider
}

// NewFakeTracer creates a FakeTracer backed by an in-memory exporter.
// Spans are available immediately after span.End() via Spans().
func NewFakeTracer() *FakeTracer {
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	return &FakeTracer{exporter: exp, provider: tp}
}

// Tracer returns a trace.Tracer from the in-memory provider.
func (f *FakeTracer) Tracer(name string) trace.Tracer {
	return f.provider.Tracer(name)
}

// Spans returns all completed spans collected in memory.
func (f *FakeTracer) Spans() tracetest.SpanStubs {
	return f.exporter.GetSpans()
}

// Reset clears collected spans.
func (f *FakeTracer) Reset() {
	f.exporter.Reset()
}

// Shutdown shuts down the underlying TracerProvider, releasing resources.
func (f *FakeTracer) Shutdown(ctx context.Context) error {
	return f.provider.Shutdown(ctx)
}
