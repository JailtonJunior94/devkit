package o11y_test

import (
	"context"
	"fmt"

	"devkit/pkg/o11y"
	"devkit/pkg/o11y/otlpgrpc"
)

func ExampleNew() {
	sdk, err := o11y.New(context.Background(), o11y.Config{
		ServiceName:    "my-service",
		ServiceVersion: "1.0.0",
		Environment:    "production",
	})
	if err != nil {
		panic(err)
	}
	defer sdk.Shutdown(context.Background()) //nolint:errcheck

	// Providers are safe to use immediately.
	_ = sdk.TracerProvider()
	_ = sdk.MeterProvider()
	_ = sdk.Logger()

	fmt.Println("o11y SDK initialized")
	// Output: o11y SDK initialized
}

func ExampleNew_otlpGRPC() {
	sdk, err := o11y.New(
		context.Background(),
		o11y.Config{
			ServiceName:    "my-service",
			ServiceVersion: "1.0.0",
			Environment:    "production",
		},
		otlpgrpc.WithTrace("localhost:4317"),
		otlpgrpc.WithMetric("localhost:4317"),
		otlpgrpc.WithLog("localhost:4317"),
	)
	if err != nil {
		panic(err)
	}
	defer sdk.Shutdown(context.Background()) //nolint:errcheck

	fmt.Println("o11y OTLP gRPC options initialized")
	// Output: o11y OTLP gRPC options initialized
}
