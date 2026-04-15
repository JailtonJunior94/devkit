// Package oteltest provides in-memory test doubles for OpenTelemetry signals.
package oteltest

import (
	"context"
	"sync"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// FakeTracer provides an in-memory TracerProvider for test span inspection.
type FakeTracer struct {
	exporter *tracetest.InMemoryExporter
	provider *sdktrace.TracerProvider
	once     sync.Once
}

// NewFakeTracer creates a FakeTracer backed by an in-memory exporter.
func NewFakeTracer() *FakeTracer {
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	return &FakeTracer{exporter: exp, provider: tp}
}

// TracerProvider returns the underlying tracer provider.
func (f *FakeTracer) TracerProvider() trace.TracerProvider {
	return f.provider
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

// Shutdown shuts down the underlying TracerProvider.
func (f *FakeTracer) Shutdown(ctx context.Context) error {
	var err error
	f.once.Do(func() {
		err = f.provider.Shutdown(ctx)
	})
	return err
}
