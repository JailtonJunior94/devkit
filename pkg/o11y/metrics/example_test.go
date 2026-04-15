package metrics_test

import (
	"context"
	"fmt"

	"devkit/pkg/o11y/metrics"
)

func ExampleNew() {
	provider, err := metrics.New(context.Background(), metrics.Config{
		ServiceName: "checkout",
	})
	if err != nil {
		panic(err)
	}
	defer provider.Shutdown(context.Background()) //nolint:errcheck

	_ = provider.MeterProvider()

	fmt.Println("metrics initialized")
	// Output: metrics initialized
}
