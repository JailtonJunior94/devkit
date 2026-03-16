package tracing_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"devkit/o11y/tracing"
)

// recordingExporter is a SpanExporter that stores spans for inspection.
// Unlike tracetest.InMemoryExporter, Shutdown does not reset the recorded spans,
// making it safe to use with WithBatcher (which calls Shutdown on the exporter
// during provider shutdown).
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

func (e *recordingExporter) Shutdown(_ context.Context) error { return nil }

func (e *recordingExporter) count() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.spans)
}

func (e *recordingExporter) spanNames() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	names := make([]string, len(e.spans))
	for i, s := range e.spans {
		names[i] = s.Name()
	}
	return names
}

func TestNew_errorOnEmptyServiceName(t *testing.T) {
	t.Parallel()

	_, err := tracing.New(context.Background(), tracing.Config{})
	if !errors.Is(err, tracing.ErrServiceNameRequired) {
		t.Errorf("New() error = %v, want ErrServiceNameRequired", err)
	}
}

func TestNew_noopWhenNoExporter(t *testing.T) {
	t.Parallel()

	p, err := tracing.New(context.Background(), tracing.Config{ServiceName: "svc"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if p == nil {
		t.Fatal("New() returned nil provider")
	}
	if p.TracerProvider() == nil {
		t.Fatal("TracerProvider() returned nil")
	}
}

func TestNew_withInMemoryExporter(t *testing.T) {
	t.Parallel()

	// Verify that tracing.New with a SpanExporter records and flushes spans.
	// Note: tracetest.InMemoryExporter.Shutdown() resets its internal state, so
	// we use recordingExporter which does not clear spans on Shutdown — making it
	// safe to inspect after p.Shutdown() (which triggers the batcher flush).
	exp := &recordingExporter{}
	p, err := tracing.New(context.Background(), tracing.Config{
		ServiceName:  "svc",
		SpanExporter: exp,
		Sampler:      sdktrace.AlwaysSample(),
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	tracer := p.TracerProvider().Tracer("test")
	_, span := tracer.Start(context.Background(), "create-order")
	span.End()

	// Shutdown flushes the batcher, delivering spans to the exporter.
	if err := p.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	if exp.count() == 0 {
		t.Fatal("expected at least one span after Shutdown flush, got none")
	}
	if names := exp.spanNames(); names[0] != "create-order" {
		t.Errorf("span name = %q, want %q", names[0], "create-order")
	}
}

func TestNew_withExporterFlush(t *testing.T) {
	t.Parallel()

	// Verify tracing.New accepts a SpanExporter and the provider works end-to-end.
	exp := tracetest.NewInMemoryExporter()
	p, err := tracing.New(context.Background(), tracing.Config{
		ServiceName:  "svc",
		SpanExporter: exp,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	tp := p.TracerProvider()
	if tp == nil {
		t.Fatal("TracerProvider() returned nil")
	}
	if err := p.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}

func TestNew_withVersion(t *testing.T) {
	t.Parallel()

	exp := tracetest.NewInMemoryExporter()
	p, err := tracing.New(context.Background(), tracing.Config{
		ServiceName:    "svc",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		SpanExporter:   exp,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := p.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}

func TestNew_withCustomSampler(t *testing.T) {
	t.Parallel()

	exp := tracetest.NewInMemoryExporter()
	p, err := tracing.New(context.Background(), tracing.Config{
		ServiceName:  "svc",
		SpanExporter: exp,
		Sampler:      sdktrace.NeverSample(),
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	_ = p.Shutdown(context.Background())
}

func TestShutdown_idempotent(t *testing.T) {
	t.Parallel()

	p, err := tracing.New(context.Background(), tracing.Config{ServiceName: "svc"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := p.Shutdown(context.Background()); err != nil {
		t.Errorf("first Shutdown() error = %v", err)
	}
	if err := p.Shutdown(context.Background()); err != nil {
		t.Errorf("second Shutdown() error = %v (should be idempotent)", err)
	}
}

func TestNew_withSamplerOption(t *testing.T) {
	t.Parallel()

	exp := tracetest.NewInMemoryExporter()
	p, err := tracing.New(
		context.Background(),
		tracing.Config{
			ServiceName:  "svc",
			SpanExporter: exp,
		},
		tracing.WithSampler(sdktrace.AlwaysSample()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	_ = p.Shutdown(context.Background())
}

func TestProvider_tracerProviderInterface(t *testing.T) {
	t.Parallel()

	p, err := tracing.New(context.Background(), tracing.Config{ServiceName: "svc"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	tp := p.TracerProvider()
	// Must satisfy trace.TracerProvider interface (compile-time already, but we
	// verify the returned value is usable).
	tracer := tp.Tracer("test")
	if tracer == nil {
		t.Error("Tracer() returned nil")
	}
}

func TestWithOTLPGRPC_noEndpoint(t *testing.T) {
	t.Parallel()

	p, err := tracing.New(context.Background(), tracing.Config{ServiceName: "svc"}, tracing.WithOTLPGRPC())
	if err != nil {
		t.Fatalf("New() with WithOTLPGRPC() error = %v", err)
	}
	_ = p.Shutdown(context.Background())
}

func TestWithOTLPGRPC_withEndpoint(t *testing.T) {
	t.Parallel()

	p, err := tracing.New(context.Background(), tracing.Config{ServiceName: "svc"}, tracing.WithOTLPGRPC("localhost:4317"))
	if err != nil {
		t.Fatalf("New() with WithOTLPGRPC(endpoint) error = %v", err)
	}
	_ = p.Shutdown(context.Background())
}

func TestWithOTLPHTTP_noEndpoint(t *testing.T) {
	t.Parallel()

	p, err := tracing.New(context.Background(), tracing.Config{ServiceName: "svc"}, tracing.WithOTLPHTTP())
	if err != nil {
		t.Fatalf("New() with WithOTLPHTTP() error = %v", err)
	}
	_ = p.Shutdown(context.Background())
}

func TestWithOTLPHTTP_withEndpoint(t *testing.T) {
	t.Parallel()

	p, err := tracing.New(context.Background(), tracing.Config{ServiceName: "svc"}, tracing.WithOTLPHTTP("localhost:4318"))
	if err != nil {
		t.Fatalf("New() with WithOTLPHTTP(endpoint) error = %v", err)
	}
	_ = p.Shutdown(context.Background())
}

func BenchmarkNew_noop(b *testing.B) {
	cfg := tracing.Config{ServiceName: "bench-svc"}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		p, _ := tracing.New(context.Background(), cfg)
		_ = p.Shutdown(context.Background())
	}
}
