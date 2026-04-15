package otlphttp

import (
	"context"
	"testing"

	"devkit/pkg/o11y/tracing"
)

func TestWithOTLPHTTPSetsSpanExporter(t *testing.T) {
	t.Parallel()

	cfg := tracing.Config{ServiceName: "svc"}
	if err := WithOTLPHTTP("localhost:4318")(context.Background(), &cfg); err != nil {
		t.Fatalf("WithOTLPHTTP() error = %v", err)
	}
	if cfg.SpanExporter == nil {
		t.Fatal("WithOTLPHTTP() did not set SpanExporter")
	}
}
