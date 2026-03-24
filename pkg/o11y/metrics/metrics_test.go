package metrics_test

import (
	"context"
	"errors"
	"testing"
	"time"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"devkit/pkg/o11y/metrics"
)

// noopMetricExporter implements sdkmetric.Exporter, discarding all data.
// Used to exercise the non-nil exporter code path without a real endpoint.
type noopMetricExporter struct{}

func (noopMetricExporter) Temporality(sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.CumulativeTemporality
}
func (noopMetricExporter) Aggregation(sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return sdkmetric.DefaultAggregationSelector(sdkmetric.InstrumentKindCounter)
}
func (noopMetricExporter) Export(_ context.Context, _ *metricdata.ResourceMetrics) error {
	return nil
}
func (noopMetricExporter) ForceFlush(_ context.Context) error { return nil }
func (noopMetricExporter) Shutdown(_ context.Context) error   { return nil }

func TestNew_errorOnEmptyServiceName(t *testing.T) {
	t.Parallel()

	_, err := metrics.New(context.Background(), metrics.Config{})
	if !errors.Is(err, metrics.ErrServiceNameRequired) {
		t.Errorf("New() error = %v, want ErrServiceNameRequired", err)
	}
}

func TestNew_noopWhenNoExporter(t *testing.T) {
	t.Parallel()

	p, err := metrics.New(context.Background(), metrics.Config{ServiceName: "svc"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if p == nil {
		t.Fatal("New() returned nil")
	}
	if p.MeterProvider() == nil {
		t.Fatal("MeterProvider() returned nil")
	}
	if err := p.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}
}

func TestNew_withVersionAndEnvironment(t *testing.T) {
	t.Parallel()

	p, err := metrics.New(context.Background(), metrics.Config{
		ServiceName:    "svc",
		ServiceVersion: "1.0.0",
		Environment:    "test",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := p.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}
}

func TestNew_withNoopExporter(t *testing.T) {
	t.Parallel()

	p, err := metrics.New(context.Background(), metrics.Config{
		ServiceName:    "svc",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		Exporter:       noopMetricExporter{},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if p.MeterProvider() == nil {
		t.Fatal("MeterProvider() returned nil")
	}

	meter := p.MeterProvider().Meter("test")
	counter, err := meter.Int64Counter("requests")
	if err != nil {
		t.Fatalf("Int64Counter() error = %v", err)
	}
	counter.Add(context.Background(), 1)

	if err := p.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}
}

func TestNew_withInterval(t *testing.T) {
	t.Parallel()

	p, err := metrics.New(context.Background(), metrics.Config{
		ServiceName: "svc",
		Exporter:    noopMetricExporter{},
		Interval:    100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := p.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}
}

func TestWithInterval_option(t *testing.T) {
	t.Parallel()

	p, err := metrics.New(
		context.Background(),
		metrics.Config{ServiceName: "svc", Exporter: noopMetricExporter{}},
		metrics.WithInterval(50*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("New() with WithInterval() error = %v", err)
	}
	if err := p.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}
}

func TestShutdown_idempotent(t *testing.T) {
	t.Parallel()

	p, err := metrics.New(context.Background(), metrics.Config{ServiceName: "svc"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := p.Shutdown(context.Background()); err != nil {
		t.Errorf("first Shutdown() error = %v", err)
	}
	if err := p.Shutdown(context.Background()); err != nil {
		t.Errorf("second Shutdown() error = %v (should be idempotent)", err)
	}
}

func BenchmarkNew_noop(b *testing.B) {
	cfg := metrics.Config{ServiceName: "bench-svc"}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		p, _ := metrics.New(context.Background(), cfg)
		_ = p.Shutdown(context.Background())
	}
}
