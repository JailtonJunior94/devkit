package oteltest_test

import (
	"context"
	"log/slog"
	"testing"

	"devkit/pkg/o11y/oteltest"
)

// --- FakeTracer tests ---

func TestFakeTracer_collectsSpans(t *testing.T) {
	t.Parallel()

	ft := oteltest.NewFakeTracer()
	tracer := ft.Tracer("test")

	_, span := tracer.Start(context.Background(), "op")
	span.End()

	spans := ft.Spans()
	if len(spans) != 1 {
		t.Errorf("Spans() = %d, want 1", len(spans))
	}
}

func TestFakeTracer_reset(t *testing.T) {
	t.Parallel()

	ft := oteltest.NewFakeTracer()
	tracer := ft.Tracer("test")

	_, span := tracer.Start(context.Background(), "op")
	span.End()
	ft.Reset()

	if len(ft.Spans()) != 0 {
		t.Error("Reset() did not clear spans")
	}
}

func TestFakeTracer_multipleSpans(t *testing.T) {
	t.Parallel()

	ft := oteltest.NewFakeTracer()
	tracer := ft.Tracer("test")

	for i := 0; i < 3; i++ {
		_, span := tracer.Start(context.Background(), "op")
		span.End()
	}

	if len(ft.Spans()) != 3 {
		t.Errorf("Spans() = %d, want 3", len(ft.Spans()))
	}
}

// --- FakeMeter tests ---

func TestFakeMeter_collectsMetrics(t *testing.T) {
	t.Parallel()

	fm := oteltest.NewFakeMeter()
	meter := fm.MeterProvider().Meter("test")

	counter, err := meter.Int64Counter("hits")
	if err != nil {
		t.Fatalf("Int64Counter() error = %v", err)
	}
	counter.Add(context.Background(), 7)

	rm, err := fm.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if len(rm.ScopeMetrics) == 0 {
		t.Error("expected scope metrics, got none")
	}
}

func TestFakeMeter_meterProviderNotNil(t *testing.T) {
	t.Parallel()

	fm := oteltest.NewFakeMeter()
	if fm.MeterProvider() == nil {
		t.Error("MeterProvider() returned nil")
	}
}

// --- FakeLogger tests ---

func TestFakeLogger_collectsRecords(t *testing.T) {
	t.Parallel()

	fl := oteltest.NewFakeLogger()
	logger := fl.Logger()

	logger.Info("hello", "key", "val")
	logger.Warn("world")

	records := fl.Records()
	if len(records) != 2 {
		t.Errorf("Records() = %d, want 2", len(records))
	}
}

func TestFakeLogger_reset(t *testing.T) {
	t.Parallel()

	fl := oteltest.NewFakeLogger()
	fl.Logger().Info("msg")
	fl.Reset()

	if len(fl.Records()) != 0 {
		t.Error("Reset() did not clear records")
	}
}

func TestFakeLogger_levels(t *testing.T) {
	t.Parallel()

	fl := oteltest.NewFakeLogger()
	logger := fl.Logger()

	logger.Debug("debug msg")
	logger.Info("info msg")
	logger.Warn("warn msg")
	logger.Error("error msg")

	records := fl.Records()
	if len(records) != 4 {
		t.Errorf("Records() = %d, want 4", len(records))
	}
}

func TestFakeLogger_messageContent(t *testing.T) {
	t.Parallel()

	fl := oteltest.NewFakeLogger()
	fl.Logger().Info("hello world")

	records := fl.Records()
	if len(records) != 1 {
		t.Fatalf("Records() = %d, want 1", len(records))
	}
	if records[0].Message != "hello world" {
		t.Errorf("Message = %q, want %q", records[0].Message, "hello world")
	}
	if records[0].Level != slog.LevelInfo {
		t.Errorf("Level = %v, want %v", records[0].Level, slog.LevelInfo)
	}
}

func TestFakeLogger_withAttrs(t *testing.T) {
	t.Parallel()

	fl := oteltest.NewFakeLogger()
	logger := fl.Logger().With("service", "svc")
	logger.Info("msg")

	records := fl.Records()
	if len(records) != 1 {
		t.Fatalf("Records() = %d, want 1", len(records))
	}
}

func TestFakeLogger_withGroup(t *testing.T) {
	t.Parallel()

	fl := oteltest.NewFakeLogger()
	// WithGroup returns a new handler; records should still be collected.
	logger := fl.Logger().WithGroup("mygroup")
	logger.Info("grouped msg")

	records := fl.Records()
	if len(records) != 1 {
		t.Fatalf("Records() = %d, want 1 after WithGroup", len(records))
	}
}

func TestFakeTracer_shutdown(t *testing.T) {
	t.Parallel()

	ft := oteltest.NewFakeTracer()
	if err := ft.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown() error = %v, want nil", err)
	}
}

func TestFakeTracer_shutdownIdempotent(t *testing.T) {
	t.Parallel()

	ft := oteltest.NewFakeTracer()
	if err := ft.Shutdown(context.Background()); err != nil {
		t.Errorf("first Shutdown() error = %v, want nil", err)
	}
	if err := ft.Shutdown(context.Background()); err != nil {
		t.Errorf("second Shutdown() error = %v, want nil (must be idempotent)", err)
	}
}

func TestFakeLogger_withGroupAndAttrs(t *testing.T) {
	t.Parallel()

	fl := oteltest.NewFakeLogger()
	// WithGroup returns a new logger scoped under "g"; logging with inline attrs
	// exercises Handle with groupPrefix != "" and inline attrs calling attrsToAny.
	logger := fl.Logger().WithGroup("g")
	logger.Info("msg", "k", "v")

	records := fl.Records()
	if len(records) != 1 {
		t.Errorf("Records() = %d, want 1", len(records))
	}
}

func TestFakeLogger_withGroupPreservesPreGroupAttrs(t *testing.T) {
	t.Parallel()

	fl := oteltest.NewFakeLogger()
	// Attrs added via With() before WithGroup() must appear outside the group,
	// matching the slog.Handler contract from the stdlib.
	logger := fl.Logger().With("pre", "outer").WithGroup("g")
	logger.Info("msg", "k", "inner")

	records := fl.Records()
	if len(records) != 1 {
		t.Fatalf("Records() = %d, want 1", len(records))
	}

	record := records[0]
	var foundPre, foundGroup bool
	record.Attrs(func(a slog.Attr) bool {
		if a.Key == "pre" {
			foundPre = true
		}
		if a.Key == "g" {
			foundGroup = true
		}
		return true
	})
	if !foundPre {
		t.Error("WithGroup must preserve pre-group attribute 'pre' at outer scope")
	}
	if !foundGroup {
		t.Error("WithGroup must create a group attribute 'g' for in-group attrs")
	}
}

func TestFakeMeter_shutdown(t *testing.T) {
	t.Parallel()

	fm := oteltest.NewFakeMeter()
	if err := fm.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown() error = %v, want nil", err)
	}
}

func TestFakeMeter_shutdownIdempotent(t *testing.T) {
	t.Parallel()

	fm := oteltest.NewFakeMeter()
	if err := fm.Shutdown(context.Background()); err != nil {
		t.Errorf("first Shutdown() error = %v, want nil", err)
	}
	if err := fm.Shutdown(context.Background()); err != nil {
		t.Errorf("second Shutdown() error = %v, want nil (must be idempotent)", err)
	}
}

func BenchmarkFakeTracer(b *testing.B) {
	ft := oteltest.NewFakeTracer()
	tracer := ft.Tracer("bench")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, span := tracer.Start(context.Background(), "op")
		span.End()
	}
}

func BenchmarkFakeMeter(b *testing.B) {
	fm := oteltest.NewFakeMeter()
	meter := fm.MeterProvider().Meter("bench")
	counter, _ := meter.Int64Counter("hits")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		counter.Add(context.Background(), 1)
	}
}

func BenchmarkFakeLogger(b *testing.B) {
	fl := oteltest.NewFakeLogger()
	logger := fl.Logger()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		logger.Info("msg", "k", "v")
	}
}
