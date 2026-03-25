package otlphttp

import (
	"context"
	"testing"

	"devkit/pkg/o11y/metrics"
)

func TestWithOTLPHTTP_setsMetricExporter(t *testing.T) {
	t.Parallel()

	cfg := metrics.Config{ServiceName: "svc"}
	if err := WithOTLPHTTP("localhost:4318")(context.Background(), &cfg); err != nil {
		t.Fatalf("WithOTLPHTTP() error = %v", err)
	}
	if cfg.Exporter == nil {
		t.Fatal("WithOTLPHTTP() did not set Exporter")
	}
}
