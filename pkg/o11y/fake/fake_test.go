package fake

import (
	"context"
	"errors"
	"testing"

	observability "devkit/pkg/o11y"
)

type customError struct{}

func (*customError) Error() string { return "boom" }

func TestFakeTracerStartInjectsSpanIntoContext(t *testing.T) {
	t.Parallel()

	tracer := NewFakeTracer()

	ctx, span := tracer.Start(nilContext(), "operation", observability.WithAttributes(observability.String("component", "test")))
	fromContext := tracer.SpanFromContext(ctx)

	if fromContext != span {
		t.Fatalf("expected span from context to match started span")
	}

	fakeSpan, ok := span.(*FakeSpan)
	if !ok {
		t.Fatalf("expected fake span type, got %T", span)
	}

	if fakeSpan.Name != "operation" {
		t.Fatalf("expected span name to be preserved, got %q", fakeSpan.Name)
	}

	if len(fakeSpan.Attributes) != 1 || fakeSpan.Attributes[0].Key != "component" {
		t.Fatalf("expected initial attributes to be preserved, got %#v", fakeSpan.Attributes)
	}
}

func TestFakeTracerContextWithSpanInjectsSpan(t *testing.T) {
	t.Parallel()

	tracer := NewFakeTracer()
	span := &FakeSpan{Name: "manual"}

	ctx := tracer.ContextWithSpan(context.Background(), span)
	if got := tracer.SpanFromContext(ctx); got != span {
		t.Fatalf("expected context to return injected span")
	}
}

func TestFakeLoggerWithDoesNotMutateParentFields(t *testing.T) {
	t.Parallel()

	logger := NewFakeLogger()
	child := logger.With(observability.String("scope", "child"))

	logger.Info(context.Background(), "parent")
	child.Info(context.Background(), "child")

	entries := logger.GetEntries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 log entries, got %d", len(entries))
	}

	if len(entries[0].Fields) != 0 {
		t.Fatalf("expected parent logger entry to remain field-free, got %#v", entries[0].Fields)
	}

	if len(entries[1].Fields) != 1 || entries[1].Fields[0].Key != "scope" {
		t.Fatalf("expected child logger entry to contain its scoped field, got %#v", entries[1].Fields)
	}
}

func TestFakeLoggerWithDoesNotCorruptFieldsBetweenEntries(t *testing.T) {
	t.Parallel()

	logger := NewFakeLogger()
	child := logger.With(observability.String("scope", "child"))

	child.Info(context.Background(), "first", observability.String("a", "1"))
	child.Info(context.Background(), "second", observability.String("b", "2"))

	entries := child.(*FakeLogger).GetEntries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	firstFields := entries[0].Fields
	secondFields := entries[1].Fields

	if len(firstFields) != 2 || firstFields[1].Key != "a" || firstFields[1].Value != "1" {
		t.Fatalf("first entry fields corrupted: %#v", firstFields)
	}

	if len(secondFields) != 2 || secondFields[1].Key != "b" || secondFields[1].Value != "2" {
		t.Fatalf("second entry fields corrupted: %#v", secondFields)
	}
}

func TestFakeMetricsGaugeRejectsNilCallback(t *testing.T) {
	t.Parallel()

	metrics := NewFakeMetrics()
	err := metrics.Gauge("queue_depth", "queue depth", "1", nil)
	if !errors.Is(err, observability.ErrNilGaugeCallback) {
		t.Fatalf("Gauge() error = %v, want ErrNilGaugeCallback", err)
	}
}

func TestFakeMetricsGaugeStoresGaugeInMemory(t *testing.T) {
	t.Parallel()

	metrics := NewFakeMetrics()
	if err := metrics.Gauge("queue_depth", "queue depth", "1", func(ctx context.Context) float64 {
		return 42
	}); err != nil {
		t.Fatalf("Gauge() error = %v", err)
	}

	gauge := metrics.GetGauge("queue_depth")
	if gauge == nil {
		t.Fatal("expected gauge to be stored")
	}

	if got := gauge.Observe(context.Background()); got != 42 {
		t.Fatalf("Observe() = %v, want 42", got)
	}
}

func TestFakeLoggerGetEntriesReturnsDetachedCopy(t *testing.T) {
	t.Parallel()

	logger := NewFakeLogger()
	logger.Info(context.Background(), "hello", observability.String("scope", "parent"))

	entries := logger.GetEntries()
	entries[0].Fields[0] = observability.String("scope", "mutated")

	fresh := logger.GetEntries()
	if fresh[0].Fields[0] != (observability.String("scope", "parent")) {
		t.Fatalf("expected stored entry to remain unchanged, got %#v", fresh[0].Fields)
	}
}

func TestFakeTracerGetSpansReturnsDetachedCopy(t *testing.T) {
	t.Parallel()

	tracer := NewFakeTracer()
	_, span := tracer.Start(context.Background(), "op", observability.WithAttributes(observability.String("scope", "parent")))

	snapshot := tracer.GetSpans()
	snapshot[0].Name = "mutated"
	snapshot[0].Attributes[0] = observability.String("scope", "mutated")

	fakeSpan, ok := span.(*FakeSpan)
	if !ok {
		t.Fatalf("expected fake span type, got %T", span)
	}
	fakeSpan.SetAttributes(observability.String("another", "field"))

	fresh := tracer.GetSpans()
	if fresh[0].Name != "op" {
		t.Fatalf("expected stored span name to remain unchanged, got %q", fresh[0].Name)
	}
	if fresh[0].Attributes[0] != (observability.String("scope", "parent")) {
		t.Fatalf("expected stored attributes to remain unchanged, got %#v", fresh[0].Attributes)
	}
}

func TestFakeMetricsGetValuesReturnsDetachedCopy(t *testing.T) {
	t.Parallel()

	metrics := NewFakeMetrics()
	counter, err := metrics.Counter("orders", "orders", "1")
	if err != nil {
		t.Fatalf("Counter() error = %v", err)
	}

	counter.Add(context.Background(), 1, observability.String("result", "success"))

	stored := metrics.GetCounter("orders")
	values := stored.GetValues()
	values[0].Fields[0] = observability.String("result", "mutated")

	fresh := stored.GetValues()
	if fresh[0].Fields[0] != (observability.String("result", "success")) {
		t.Fatalf("expected stored metric fields to remain unchanged, got %#v", fresh[0].Fields)
	}
}

func TestFakeSpanRecordErrorIgnoresTypedNil(t *testing.T) {
	t.Parallel()

	var typedNil *customError

	span := &FakeSpan{}
	span.RecordError(typedNil)

	if span.RecordedErr != nil {
		t.Fatalf("expected typed nil error to be ignored, got %v", span.RecordedErr)
	}
}

func nilContext() context.Context {
	return nil
}
