package logging_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	sdklog "go.opentelemetry.io/otel/sdk/log"

	"devkit/o11y/logging"
	"devkit/o11y/oteltest"
)

// noopLogExporter is a minimal sdklog.Exporter that discards all records,
// used to exercise the non-nil LogExporter code path without a real endpoint.
type noopLogExporter struct{}

func (noopLogExporter) Export(_ context.Context, _ []sdklog.Record) error { return nil }
func (noopLogExporter) Shutdown(_ context.Context) error                  { return nil }
func (noopLogExporter) ForceFlush(_ context.Context) error                { return nil }

func TestNew_errorOnEmptyServiceName(t *testing.T) {
	t.Parallel()

	_, err := logging.New(context.Background(), logging.Config{})
	if !errors.Is(err, logging.ErrServiceNameRequired) {
		t.Errorf("New() error = %v, want ErrServiceNameRequired", err)
	}
}

func TestNew_fallbackWhenNoExporter(t *testing.T) {
	t.Parallel()

	p, err := logging.New(context.Background(), logging.Config{ServiceName: "svc"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if p == nil {
		t.Fatal("New() returned nil")
	}
	if p.Logger() == nil {
		t.Fatal("Logger() returned nil")
	}
}

func TestNew_withVersionAndEnvironment(t *testing.T) {
	t.Parallel()

	p, err := logging.New(context.Background(), logging.Config{
		ServiceName:    "svc",
		ServiceVersion: "1.0.0",
		Environment:    "staging",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := p.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}
}

func TestNew_withNoopExporter(t *testing.T) {
	t.Parallel()

	p, err := logging.New(context.Background(), logging.Config{
		ServiceName:    "svc",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		LogExporter:    noopLogExporter{},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if p.Logger() == nil {
		t.Fatal("Logger() returned nil")
	}
	p.Logger().Info("test message", "k", "v")
	if err := p.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}
}

func TestShutdown_idempotent(t *testing.T) {
	t.Parallel()

	p, err := logging.New(context.Background(), logging.Config{ServiceName: "svc"})
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

func TestLogger_isUsable(t *testing.T) {
	t.Parallel()

	p, err := logging.New(context.Background(), logging.Config{ServiceName: "svc"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer p.Shutdown(context.Background()) //nolint:errcheck

	logger := p.Logger()
	// Log without panicking — basic usability check.
	logger.Info("test log", "key", "value")
}

func TestLogger_traceCorrelation(t *testing.T) {
	t.Parallel()

	// Arrange: capturing exporter that stores records.
	cap := &captureLogExporter{}
	p, err := logging.New(context.Background(), logging.Config{
		ServiceName: "svc",
		LogExporter: cap,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Create a real span so the context carries a valid TraceID.
	ft := oteltest.NewFakeTracer()
	tracer := ft.Tracer("test")
	spanCtx, span := tracer.Start(context.Background(), "op")
	defer span.End()

	// Act: log within the active span context.
	p.Logger().InfoContext(spanCtx, "msg")

	// Shutdown flushes the BatchProcessor.
	if err := p.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	// Assert: at least one record must carry a valid TraceID.
	records := cap.exported()
	if len(records) == 0 {
		t.Fatal("expected at least one exported log record, got none")
	}
	if !records[0].TraceID().IsValid() {
		t.Error("TraceID is not valid; expected trace correlation to be injected")
	}
}

// captureLogExporter collects sdklog.Record values for assertion in tests.
type captureLogExporter struct {
	mu      sync.Mutex
	records []sdklog.Record
}

func (c *captureLogExporter) Export(_ context.Context, records []sdklog.Record) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.records = append(c.records, records...)
	return nil
}

func (c *captureLogExporter) Shutdown(_ context.Context) error  { return nil }
func (c *captureLogExporter) ForceFlush(_ context.Context) error { return nil }

func (c *captureLogExporter) exported() []sdklog.Record {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make([]sdklog.Record, len(c.records))
	copy(result, c.records)
	return result
}

func BenchmarkNew_noop(b *testing.B) {
	cfg := logging.Config{ServiceName: "bench-svc"}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		p, _ := logging.New(context.Background(), cfg)
		_ = p.Shutdown(context.Background())
	}
}
