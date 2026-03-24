package oteltest_test

import (
	"context"
	"fmt"

	"devkit/pkg/o11y/oteltest"
)

func ExampleNewFakeTracer() {
	ft := oteltest.NewFakeTracer()
	tracer := ft.Tracer("my-service")

	_, span := tracer.Start(context.Background(), "create-order")
	span.End()

	spans := ft.Spans()
	fmt.Println(len(spans))
	fmt.Println(spans[0].Name)
	// Output:
	// 1
	// create-order
}

func ExampleFakeTracer_Reset() {
	ft := oteltest.NewFakeTracer()
	tracer := ft.Tracer("my-service")

	_, span := tracer.Start(context.Background(), "op")
	span.End()

	ft.Reset()
	fmt.Println(len(ft.Spans()))
	// Output:
	// 0
}

func ExampleNewFakeLogger() {
	fl := oteltest.NewFakeLogger()
	logger := fl.Logger()

	logger.Info("user signed in", "user_id", "u123")

	records := fl.Records()
	fmt.Println(len(records))
	fmt.Println(records[0].Message)
	// Output:
	// 1
	// user signed in
}

func ExampleNewFakeMeter() {
	fm := oteltest.NewFakeMeter()
	meter := fm.MeterProvider().Meter("my-service")

	counter, _ := meter.Int64Counter("requests_total")
	counter.Add(context.Background(), 5)

	rm, _ := fm.Collect(context.Background())
	fmt.Println(len(rm.ScopeMetrics) > 0)
	// Output:
	// true
}
