package tracing_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"devkit/pkg/o11y/tracing"
)

type recordingExporter struct {
	mu    sync.Mutex
	spans []sdktrace.ReadOnlySpan
}

func (e *recordingExporter) ExportSpans(_ context.Context, spans []sdktrace.ReadOnlySpan) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.spans = append(e.spans, spans...)
	return nil
}

func (e *recordingExporter) Shutdown(context.Context) error { return nil }

func (e *recordingExporter) Count() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.spans)
}

func TestNewReturnsErrorWhenServiceNameIsEmpty(t *testing.T) {
	t.Parallel()

	_, err := tracing.New(context.Background(), tracing.Config{})
	if !errors.Is(err, tracing.ErrServiceNameRequired) {
		t.Fatalf("New() error = %v, want ErrServiceNameRequired", err)
	}
}

func TestNewReturnsErrorWhenOptionIsNil(t *testing.T) {
	t.Parallel()

	_, err := tracing.New(context.Background(), tracing.Config{ServiceName: "svc"}, nil)
	if !errors.Is(err, tracing.ErrNilOption) {
		t.Fatalf("New() error = %v, want ErrNilOption", err)
	}
}

func TestNewReturnsNoopProviderWhenExporterIsNil(t *testing.T) {
	t.Parallel()

	provider, err := tracing.New(context.Background(), tracing.Config{ServiceName: "svc"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if provider.TracerProvider() == nil {
		t.Fatal("TracerProvider() returned nil")
	}
}

func TestNewExportsSpansOnShutdown(t *testing.T) {
	t.Parallel()

	exporter := &recordingExporter{}
	provider, err := tracing.New(context.Background(), tracing.Config{
		ServiceName:  "svc",
		SpanExporter: exporter,
		Sampler:      sdktrace.AlwaysSample(),
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, span := provider.TracerProvider().Tracer("test").Start(context.Background(), "operation")
	span.End()

	if err := provider.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if exporter.Count() == 0 {
		t.Fatal("expected spans to be exported")
	}
	if err := provider.Shutdown(context.Background()); err != nil {
		t.Fatalf("second Shutdown() error = %v", err)
	}
}
