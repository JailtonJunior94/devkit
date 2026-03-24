package tracing_test

import (
	"context"
	"fmt"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"devkit/pkg/o11y/tracing"
)

func ExampleNew_noop() {
	p, err := tracing.New(context.Background(), tracing.Config{
		ServiceName: "my-service",
	})
	if err != nil {
		panic(err)
	}
	defer p.Shutdown(context.Background()) //nolint:errcheck

	tracer := p.TracerProvider().Tracer("example")
	_, span := tracer.Start(context.Background(), "my-operation")
	span.End()

	fmt.Println("tracing initialized")
	// Output: tracing initialized
}

func ExampleNew_inMemory() {
	// Use WithSyncer for in-memory testing to get immediate span access.
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	defer tp.Shutdown(context.Background()) //nolint:errcheck

	tracer := tp.Tracer("example")
	_, span := tracer.Start(context.Background(), "my-operation")
	span.End()

	fmt.Printf("collected %d span(s)\n", len(exp.GetSpans()))
	// Output: collected 1 span(s)
}
