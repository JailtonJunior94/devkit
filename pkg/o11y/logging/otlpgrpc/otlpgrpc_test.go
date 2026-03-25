package otlpgrpc

import (
	"context"
	"testing"

	"devkit/pkg/o11y/logging"
)

func TestWithOTLPGRPC_setsLogExporter(t *testing.T) {
	t.Parallel()

	cfg := logging.Config{ServiceName: "svc"}
	if err := WithOTLPGRPC("localhost:4317")(context.Background(), &cfg); err != nil {
		t.Fatalf("WithOTLPGRPC() error = %v", err)
	}
	if cfg.LogExporter == nil {
		t.Fatal("WithOTLPGRPC() did not set LogExporter")
	}
}
