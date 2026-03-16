package oteltest

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// FakeMeter provides an in-memory MeterProvider for test metric inspection.
// Use Collect to retrieve recorded measurements.
type FakeMeter struct {
	reader   *sdkmetric.ManualReader
	provider *sdkmetric.MeterProvider
	once     sync.Once
}

// NewFakeMeter creates a FakeMeter backed by a ManualReader.
func NewFakeMeter() *FakeMeter {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	return &FakeMeter{reader: reader, provider: mp}
}

// MeterProvider returns the underlying metric.MeterProvider.
func (f *FakeMeter) MeterProvider() metric.MeterProvider {
	return f.provider
}

// Collect triggers manual collection and returns the current metric data.
func (f *FakeMeter) Collect(ctx context.Context) (metricdata.ResourceMetrics, error) {
	var rm metricdata.ResourceMetrics
	err := f.reader.Collect(ctx, &rm)
	return rm, err
}

// Shutdown shuts down the underlying MeterProvider, releasing resources.
// Idempotent: subsequent calls are no-ops and return nil.
func (f *FakeMeter) Shutdown(ctx context.Context) error {
	var err error
	f.once.Do(func() {
		err = f.provider.Shutdown(ctx)
	})
	return err
}
