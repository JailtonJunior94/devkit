package otel

import (
	"context"
	"testing"

	observability "devkit/pkg/o11y"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestRecordErrorHandlesTypedNilErrorWithoutPanic(t *testing.T) {
	t.Parallel()

	tp := sdktrace.NewTracerProvider()
	t.Cleanup(func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			t.Fatalf("shutdown tracer provider: %v", err)
		}
	})

	tracer := newOtelTracer(tp.Tracer("test"))
	ctx, span := tracer.Start(context.Background(), "op")
	_ = ctx

	var err error = (*typedNilError)(nil)

	// Must not panic on typed nil error.
	span.RecordError(err)
	span.RecordError(err, observability.String("extra", "field"))
	span.RecordError(nil)
	span.End()
}
