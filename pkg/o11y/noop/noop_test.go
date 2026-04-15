package noop

import (
	"context"
	"errors"
	"testing"

	observability "devkit/pkg/o11y"
)

func TestNoopTracerStartInjectsSpanIntoContext(t *testing.T) {
	t.Parallel()

	tracer := &noopTracer{}

	ctx, span := tracer.Start(nilContext(), "operation")
	fromContext := tracer.SpanFromContext(ctx)

	if fromContext != span {
		t.Fatalf("expected span from context to match started span")
	}
}

func TestNoopTracerContextWithSpanInjectsSpan(t *testing.T) {
	t.Parallel()

	tracer := &noopTracer{}
	span := noopSpan{}

	ctx := tracer.ContextWithSpan(nilContext(), span)
	if got := tracer.SpanFromContext(ctx); got != span {
		t.Fatalf("expected context to return injected span")
	}
}

func TestNoopMetricsGaugeRejectsNilCallback(t *testing.T) {
	t.Parallel()

	metrics := &noopMetrics{}
	err := metrics.Gauge("queue_depth", "queue depth", "1", nil)
	if !errors.Is(err, observability.ErrNilGaugeCallback) {
		t.Fatalf("Gauge() error = %v, want ErrNilGaugeCallback", err)
	}
}

func nilContext() context.Context {
	return nil
}
