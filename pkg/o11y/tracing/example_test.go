package tracing_test

import (
	"context"
	"fmt"

	"devkit/pkg/o11y/tracing"
)

func ExampleNew() {
	provider, err := tracing.New(context.Background(), tracing.Config{
		ServiceName: "checkout",
	})
	if err != nil {
		panic(err)
	}
	defer provider.Shutdown(context.Background()) //nolint:errcheck

	tracer := provider.TracerProvider().Tracer("example")
	_, span := tracer.Start(context.Background(), "operation")
	span.End()

	fmt.Println("tracing initialized")
	// Output: tracing initialized
}
