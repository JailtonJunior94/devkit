package o11y_test

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"devkit/pkg/o11y"
)

// countingSpanExporter counts exported spans without resetting on Shutdown.
type countingSpanExporter struct{ n atomic.Int32 }

func (e *countingSpanExporter) ExportSpans(_ context.Context, spans []sdktrace.ReadOnlySpan) error {
	e.n.Add(int32(len(spans)))
	return nil
}
func (e *countingSpanExporter) Shutdown(_ context.Context) error { return nil }

// noopLogExporter discards all log records.
type noopLogExporter struct{}

func (noopLogExporter) Export(_ context.Context, _ []sdklog.Record) error { return nil }
func (noopLogExporter) Shutdown(_ context.Context) error                  { return nil }
func (noopLogExporter) ForceFlush(_ context.Context) error                { return nil }

// noopMetricExporter is a minimal sdkmetric.Exporter for testing.
type noopMetricExporter struct{}

func (noopMetricExporter) Temporality(sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.CumulativeTemporality
}
func (noopMetricExporter) Aggregation(k sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return sdkmetric.DefaultAggregationSelector(k)
}
func (noopMetricExporter) Export(_ context.Context, _ *metricdata.ResourceMetrics) error { return nil }
func (noopMetricExporter) ForceFlush(_ context.Context) error                            { return nil }
func (noopMetricExporter) Shutdown(_ context.Context) error                              { return nil }

func TestNew_errorOnEmptyServiceName(t *testing.T) {
	t.Parallel()

	_, err := o11y.New(context.Background(), o11y.Config{})
	if !errors.Is(err, o11y.ErrServiceNameRequired) {
		t.Fatalf("New() error = %v, want ErrServiceNameRequired", err)
	}
}

func TestNew_noopProviders(t *testing.T) {
	t.Parallel()

	sdk, err := o11y.New(context.Background(), o11y.Config{ServiceName: "svc"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if sdk == nil {
		t.Fatal("New() returned nil SDK")
	}
	if sdk.TracerProvider() == nil {
		t.Fatal("TracerProvider() returned nil")
	}
	if sdk.MeterProvider() == nil {
		t.Fatal("MeterProvider() returned nil")
	}
	if sdk.Logger() == nil {
		t.Fatal("Logger() returned nil")
	}
}

func TestNew_withVersionAndEnvironment(t *testing.T) {
	t.Parallel()

	sdk, err := o11y.New(context.Background(), o11y.Config{
		ServiceName:    "svc",
		ServiceVersion: "1.0.0",
		Environment:    "test",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := sdk.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}
}

func TestShutdown_idempotent(t *testing.T) {
	t.Parallel()

	sdk, err := o11y.New(context.Background(), o11y.Config{ServiceName: "svc"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := sdk.Shutdown(context.Background()); err != nil {
		t.Errorf("first Shutdown() error = %v", err)
	}
	if err := sdk.Shutdown(context.Background()); err != nil {
		t.Errorf("second Shutdown() error = %v (should be idempotent)", err)
	}
}

func TestSDK_tracerProviderIsUsable(t *testing.T) {
	t.Parallel()

	sdk, err := o11y.New(context.Background(), o11y.Config{ServiceName: "svc"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer sdk.Shutdown(context.Background()) //nolint:errcheck

	tracer := sdk.TracerProvider().Tracer("test")
	_, span := tracer.Start(context.Background(), "op")
	span.End()
}

func TestSDK_meterProviderIsUsable(t *testing.T) {
	t.Parallel()

	sdk, err := o11y.New(context.Background(), o11y.Config{ServiceName: "svc"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer sdk.Shutdown(context.Background()) //nolint:errcheck

	meter := sdk.MeterProvider().Meter("test")
	counter, err := meter.Int64Counter("requests")
	if err != nil {
		t.Fatalf("Int64Counter() error = %v", err)
	}
	counter.Add(context.Background(), 1)
}

func TestSDK_loggerIsUsable(t *testing.T) {
	t.Parallel()

	sdk, err := o11y.New(context.Background(), o11y.Config{ServiceName: "svc"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer sdk.Shutdown(context.Background()) //nolint:errcheck

	sdk.Logger().Info("test", "key", "value")
}

func TestNew_withLogExporter(t *testing.T) {
	t.Parallel()

	sdk, err := o11y.New(
		context.Background(),
		o11y.Config{ServiceName: "svc"},
		o11y.WithLogExporter(noopLogExporter{}),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer sdk.Shutdown(context.Background()) //nolint:errcheck

	logger := sdk.Logger()
	if logger == nil {
		t.Fatal("Logger() returned nil")
	}
	logger.Info("test", "k", "v")
}

func TestShutdown_returnsJoinedErrors(t *testing.T) {
	t.Parallel()

	// Arrange: a log exporter whose Shutdown always returns an error.
	// The log SDK's BatchProcessor propagates the exporter's Shutdown error
	// (unlike the trace BatchSpanProcessor which calls otel.Handle and swallows it).
	wantErr := errors.New("exporter shutdown failed")
	failExp := &failingLogExporter{err: wantErr}

	sdk, err := o11y.New(
		context.Background(),
		o11y.Config{ServiceName: "svc"},
		o11y.WithLogExporter(failExp),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Act: Shutdown should surface the exporter's error.
	shutdownErr := sdk.Shutdown(context.Background())

	// Assert: the returned error must contain the exporter's message.
	if shutdownErr == nil {
		t.Fatal("Shutdown() expected an error, got nil")
	}
	if !strings.Contains(shutdownErr.Error(), wantErr.Error()) {
		t.Errorf("Shutdown() error = %v, want it to contain %q", shutdownErr, wantErr.Error())
	}
}

// failingLogExporter is an sdklog.Exporter whose Shutdown always returns an error.
type failingLogExporter struct {
	err error
}

func (f *failingLogExporter) Export(_ context.Context, _ []sdklog.Record) error { return nil }
func (f *failingLogExporter) ForceFlush(_ context.Context) error                { return nil }
func (f *failingLogExporter) Shutdown(_ context.Context) error                  { return f.err }

func TestNew_optionReturnsError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("option failed")
	failOpt := func(_ context.Context, _ *o11y.Config) error {
		return wantErr
	}

	_, err := o11y.New(context.Background(), o11y.Config{ServiceName: "svc"}, failOpt)
	if !errors.Is(err, wantErr) {
		t.Errorf("New() error = %v, want to wrap %v", err, wantErr)
	}
}

func TestNew_doesNotRegisterGlobalProvider(t *testing.T) {
	t.Parallel()

	// Record global TracerProvider before creating the SDK.
	before := otel.GetTracerProvider()

	sdk, err := o11y.New(context.Background(), o11y.Config{ServiceName: "svc"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer sdk.Shutdown(context.Background()) //nolint:errcheck

	// The global provider must not have changed.
	after := otel.GetTracerProvider()
	if before != after {
		t.Error("New() must not register a global TracerProvider")
	}
}

func TestNew_withTraceSampler(t *testing.T) {
	t.Parallel()

	sdk, err := o11y.New(
		context.Background(),
		o11y.Config{ServiceName: "svc"},
		o11y.WithSampler(sdktrace.AlwaysSample()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer sdk.Shutdown(context.Background()) //nolint:errcheck

	tracer := sdk.TracerProvider().Tracer("test")
	_, span := tracer.Start(context.Background(), "op")
	span.End()
}

func TestNew_withSpanExporter(t *testing.T) {
	t.Parallel()

	exp := tracetest.NewInMemoryExporter()
	sdk, err := o11y.New(
		context.Background(),
		o11y.Config{
			ServiceName:    "svc",
			ServiceVersion: "1.0.0",
			Environment:    "test",
		},
		o11y.WithSpanExporter(exp),
		o11y.WithSampler(sdktrace.AlwaysSample()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer sdk.Shutdown(context.Background()) //nolint:errcheck

	tp := sdk.TracerProvider()
	if tp == nil {
		t.Fatal("TracerProvider() returned nil")
	}
}

func TestNew_withMetricExporter(t *testing.T) {
	t.Parallel()

	sdk, err := o11y.New(
		context.Background(),
		o11y.Config{ServiceName: "svc"},
		o11y.WithMetricExporter(noopMetricExporter{}),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer sdk.Shutdown(context.Background()) //nolint:errcheck

	mp := sdk.MeterProvider()
	if mp == nil {
		t.Fatal("MeterProvider() returned nil")
	}
	meter := mp.Meter("test")
	counter, err := meter.Int64Counter("ops")
	if err != nil {
		t.Fatalf("Int64Counter() error = %v", err)
	}
	counter.Add(context.Background(), 1)
}

func TestNew_configFieldInjection(t *testing.T) {
	t.Parallel()

	exp := &countingSpanExporter{}
	sdk, err := o11y.New(
		context.Background(),
		o11y.Config{
			ServiceName:   "svc",
			TraceExporter: exp,
			TraceSampler:  sdktrace.AlwaysSample(),
		},
	)
	if err != nil {
		t.Fatalf("New() with Config field injection error = %v", err)
	}

	tracer := sdk.TracerProvider().Tracer("test")
	_, span := tracer.Start(context.Background(), "op")
	span.End()

	// Shutdown flushes the batcher, delivering spans to the exporter.
	if err := sdk.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}

	if exp.n.Load() == 0 {
		t.Error("expected spans to be recorded via Config.TraceExporter, got 0")
	}
}

func TestNew_withMetricInterval(t *testing.T) {
	t.Parallel()

	sdk, err := o11y.New(
		context.Background(),
		o11y.Config{ServiceName: "svc"},
		o11y.WithMetricExporter(noopMetricExporter{}),
		o11y.WithMetricInterval(100*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	sdk.Shutdown(context.Background()) //nolint:errcheck
}

func TestNew_concurrentAccess(t *testing.T) {
	t.Parallel()

	sdk, err := o11y.New(context.Background(), o11y.Config{ServiceName: "svc"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			_ = sdk.TracerProvider().Tracer("concurrent")
			_ = sdk.MeterProvider().Meter("concurrent")
			_ = sdk.Logger()
		}()
	}
	wg.Wait()
	if err := sdk.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}
}

func TestNew_concurrentShutdown(t *testing.T) {
	t.Parallel()

	sdk, err := o11y.New(context.Background(), o11y.Config{ServiceName: "svc"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	const goroutines = 5
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			_ = sdk.Shutdown(context.Background())
		}()
	}
	wg.Wait()
}

func TestWithW3CPropagators(t *testing.T) {
	// Not parallel: mutates the global OTel TextMapPropagator.
	prev := otel.GetTextMapPropagator()
	defer otel.SetTextMapPropagator(prev)

	sdk, err := o11y.New(context.Background(), o11y.Config{ServiceName: "svc"}, o11y.WithW3CPropagators())
	if err != nil {
		t.Fatalf("New() with WithW3CPropagators() error = %v", err)
	}
	defer sdk.Shutdown(context.Background()) //nolint:errcheck

	// Verify the global propagator was registered.
	prop := otel.GetTextMapPropagator()
	if prop == nil {
		t.Error("WithW3CPropagators() must register a non-nil global propagator")
	}
}

func TestNew_exposesDefaultPropagatorWithoutMutatingGlobalState(t *testing.T) {
	t.Parallel()

	before := otel.GetTextMapPropagator()
	sdk, err := o11y.New(context.Background(), o11y.Config{ServiceName: "svc"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer sdk.Shutdown(context.Background()) //nolint:errcheck

	if sdk.Propagator() == nil {
		t.Fatal("Propagator() returned nil")
	}

	carrier := propagation.MapCarrier{}
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    [16]byte{1},
		SpanID:     [8]byte{2},
		TraceFlags: trace.FlagsSampled,
	}))
	sdk.Propagator().Inject(ctx, carrier)
	if carrier.Get("traceparent") == "" {
		t.Fatal("default propagator did not inject traceparent")
	}

	after := otel.GetTextMapPropagator()
	if before != after {
		t.Fatal("New() must not mutate the global TextMapPropagator by default")
	}
}

func TestNew_withCustomHandler(t *testing.T) {
	t.Parallel()

	capture := newCaptureSlogHandler()
	sdk, err := o11y.New(context.Background(), o11y.Config{
		ServiceName: "svc",
		Handler:     capture,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer sdk.Shutdown(context.Background()) //nolint:errcheck

	sdk.Logger().Info("facade")

	if got := len(capture.records()); got != 1 {
		t.Fatalf("custom handler saw %d records, want 1", got)
	}
}

type captureSlogHandlerRoot struct {
	mu      sync.Mutex
	records []slog.Record
}

type captureSlogHandler struct {
	root *captureSlogHandlerRoot
}

func newCaptureSlogHandler() *captureSlogHandler {
	return &captureSlogHandler{root: &captureSlogHandlerRoot{}}
}

func (h *captureSlogHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *captureSlogHandler) Handle(_ context.Context, record slog.Record) error {
	h.root.mu.Lock()
	defer h.root.mu.Unlock()
	h.root.records = append(h.root.records, record.Clone())
	return nil
}

func (h *captureSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &captureSlogHandler{root: h.root}
}

func (h *captureSlogHandler) WithGroup(name string) slog.Handler {
	return &captureSlogHandler{root: h.root}
}

func (h *captureSlogHandler) records() []slog.Record {
	h.root.mu.Lock()
	defer h.root.mu.Unlock()
	out := make([]slog.Record, len(h.root.records))
	copy(out, h.root.records)
	return out
}

func BenchmarkNew_noop(b *testing.B) {
	cfg := o11y.Config{ServiceName: "bench-svc"}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sdk, _ := o11y.New(context.Background(), cfg)
		_ = sdk.Shutdown(context.Background())
	}
}
