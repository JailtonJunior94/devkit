package o11y_test

import (
	"context"
	"fmt"

	"devkit/pkg/o11y"
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
