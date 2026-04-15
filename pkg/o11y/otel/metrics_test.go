package otel

import (
	"context"
	"errors"
	"testing"

	observability "devkit/pkg/o11y"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func TestGaugeRejectsNilCallback(t *testing.T) {
	t.Parallel()

	meterProvider := sdkmetric.NewMeterProvider()
	t.Cleanup(func() {
		if err := meterProvider.Shutdown(context.Background()); err != nil {
			t.Fatalf("shutdown meter provider: %v", err)
		}
	})

	metrics := newOtelMetrics(meterProvider.Meter("test"))

	err := metrics.Gauge("queue_depth", "queue depth", "1", nil)
	if !errors.Is(err, observability.ErrNilGaugeCallback) {
		t.Fatalf("Gauge() error = %v, want ErrNilGaugeCallback", err)
	}
}
