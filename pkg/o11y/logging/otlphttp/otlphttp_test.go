package otlphttp

import (
	"context"
	"testing"

	"devkit/pkg/o11y/logging"
)

func TestWithOTLPHTTPSetsLogExporter(t *testing.T) {
	t.Parallel()

	cfg := logging.Config{ServiceName: "svc"}
	if err := WithOTLPHTTP("localhost:4318")(context.Background(), &cfg); err != nil {
		t.Fatalf("WithOTLPHTTP() error = %v", err)
	}
	if cfg.LogExporter == nil {
		t.Fatal("WithOTLPHTTP() did not set LogExporter")
	}
}
