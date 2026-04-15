package metrics_test

import (
	"context"
	"errors"
	"testing"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"devkit/pkg/o11y/metrics"
)

type metricExporter struct {
	exports int
}

func (e *metricExporter) Temporality(sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.CumulativeTemporality
}

func (e *metricExporter) Aggregation(sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return sdkmetric.DefaultAggregationSelector(sdkmetric.InstrumentKindCounter)
}

func (e *metricExporter) Export(context.Context, *metricdata.ResourceMetrics) error {
	e.exports++
	return nil
}

func (e *metricExporter) ForceFlush(context.Context) error { return nil }
func (e *metricExporter) Shutdown(context.Context) error   { return nil }

func TestNewReturnsErrorWhenServiceNameIsEmpty(t *testing.T) {
	t.Parallel()

	_, err := metrics.New(context.Background(), metrics.Config{})
	if !errors.Is(err, metrics.ErrServiceNameRequired) {
		t.Fatalf("New() error = %v, want ErrServiceNameRequired", err)
	}
}

func TestNewReturnsErrorWhenOptionIsNil(t *testing.T) {
	t.Parallel()

	_, err := metrics.New(context.Background(), metrics.Config{ServiceName: "svc"}, nil)
	if !errors.Is(err, metrics.ErrNilOption) {
		t.Fatalf("New() error = %v, want ErrNilOption", err)
	}
}

func TestNewReturnsNoopProviderWhenExporterIsNil(t *testing.T) {
	t.Parallel()

	provider, err := metrics.New(context.Background(), metrics.Config{ServiceName: "svc"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if provider.MeterProvider() == nil {
		t.Fatal("MeterProvider() returned nil")
	}
}

func TestNewExportsMetricsOnShutdown(t *testing.T) {
	t.Parallel()

	exporter := &metricExporter{}
	provider, err := metrics.New(context.Background(), metrics.Config{
		ServiceName: "svc",
		Exporter:    exporter,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	meter := provider.MeterProvider().Meter("test")
	counter, err := meter.Int64Counter("requests")
	if err != nil {
		t.Fatalf("Int64Counter() error = %v", err)
	}
	counter.Add(context.Background(), 1)

	if err := provider.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if exporter.exports == 0 {
		t.Fatal("expected metrics exporter to be called at least once")
	}
	if err := provider.Shutdown(context.Background()); err != nil {
		t.Fatalf("second Shutdown() error = %v", err)
	}
}
