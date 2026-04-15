package oteltest_test

import (
	"context"
	"testing"

	"devkit/pkg/o11y/oteltest"
)

func TestFakeTracerCollectsSpans(t *testing.T) {
	t.Parallel()

	ft := oteltest.NewFakeTracer()
	tracer := ft.Tracer("test")
	_, span := tracer.Start(context.Background(), "op")
	span.End()

	if len(ft.Spans()) != 1 {
		t.Fatalf("Spans() = %d, want 1", len(ft.Spans()))
	}
}

func TestFakeMeterCollectsMetrics(t *testing.T) {
	t.Parallel()

	fm := oteltest.NewFakeMeter()
	meter := fm.MeterProvider().Meter("test")
	counter, err := meter.Int64Counter("hits")
	if err != nil {
		t.Fatalf("Int64Counter() error = %v", err)
	}
	counter.Add(context.Background(), 1)

	rm, err := fm.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if len(rm.ScopeMetrics) == 0 {
		t.Fatal("expected collected metrics")
	}
}

func TestFakeLoggerCollectsRecords(t *testing.T) {
	t.Parallel()

	fl := oteltest.NewFakeLogger()
	fl.Logger().Info("hello", "k", "v")
	if len(fl.Records()) != 1 {
		t.Fatalf("Records() = %d, want 1", len(fl.Records()))
	}
}
