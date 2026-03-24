package metrics_test

import (
	"context"
	"fmt"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"devkit/pkg/o11y/metrics"
)

func ExampleNew_noop() {
	p, err := metrics.New(context.Background(), metrics.Config{
		ServiceName: "my-service",
	})
	if err != nil {
		panic(err)
	}
	defer p.Shutdown(context.Background()) //nolint:errcheck

	meter := p.MeterProvider().Meter("example")
	counter, _ := meter.Int64Counter("requests")
	counter.Add(context.Background(), 1)

	fmt.Println("metrics initialized")
	// Output: metrics initialized
}

func ExampleNew_manualReader() {
	// ManualReader allows test-time metric collection without an OTLP endpoint.
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	defer mp.Shutdown(context.Background()) //nolint:errcheck

	meter := mp.Meter("example")
	counter, _ := meter.Int64Counter("hits")
	counter.Add(context.Background(), 42)

	var rm metricdata.ResourceMetrics
	_ = reader.Collect(context.Background(), &rm)
	fmt.Printf("scope metrics: %d\n", len(rm.ScopeMetrics))
	// Output: scope metrics: 1
}
