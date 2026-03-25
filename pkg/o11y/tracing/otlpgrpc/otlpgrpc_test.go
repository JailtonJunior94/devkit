package otlpgrpc

import (
	"context"
	"testing"

	"devkit/pkg/o11y/tracing"
)

func TestWithOTLPGRPC_setsSpanExporter(t *testing.T) {
	t.Parallel()

	cfg := tracing.Config{ServiceName: "svc"}
	if err := WithOTLPGRPC("localhost:4317")(context.Background(), &cfg); err != nil {
		t.Fatalf("WithOTLPGRPC() error = %v", err)
	}
	if cfg.SpanExporter == nil {
		t.Fatal("WithOTLPGRPC() did not set SpanExporter")
	}
}
