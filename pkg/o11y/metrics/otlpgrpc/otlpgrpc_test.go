package otlpgrpc

import (
	"context"
	"testing"

	"devkit/pkg/o11y/metrics"
)

func TestWithOTLPGRPCSetsMetricExporter(t *testing.T) {
	t.Parallel()

	cfg := metrics.Config{ServiceName: "svc"}
	if err := WithOTLPGRPC("localhost:4317")(context.Background(), &cfg); err != nil {
		t.Fatalf("WithOTLPGRPC() error = %v", err)
	}
	if cfg.Exporter == nil {
		t.Fatal("WithOTLPGRPC() did not set Exporter")
	}
}
