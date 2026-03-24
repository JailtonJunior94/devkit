package otlpgrpc_test

import (
	"context"
	"testing"

	"devkit/pkg/o11y"
	"devkit/pkg/o11y/otlpgrpc"
)

func TestWithTrace_noEndpoint(t *testing.T) {
	t.Parallel()

	sdk, err := o11y.New(
		context.Background(),
		o11y.Config{ServiceName: "svc"},
		otlpgrpc.WithTrace(),
	)
	if err != nil {
		t.Fatalf("New() with WithTrace() error = %v", err)
	}
	_ = sdk.Shutdown(context.Background())
}

func TestWithTrace_withEndpoint(t *testing.T) {
	t.Parallel()

	sdk, err := o11y.New(
		context.Background(),
		o11y.Config{ServiceName: "svc"},
		otlpgrpc.WithTrace("localhost:4317"),
	)
	if err != nil {
		t.Fatalf("New() with WithTrace(endpoint) error = %v", err)
	}
	_ = sdk.Shutdown(context.Background())
}

func TestWithMetric_noEndpoint(t *testing.T) {
	t.Parallel()

	sdk, err := o11y.New(
		context.Background(),
		o11y.Config{ServiceName: "svc"},
		otlpgrpc.WithMetric(),
	)
	if err != nil {
		t.Fatalf("New() with WithMetric() error = %v", err)
	}
	_ = sdk.Shutdown(context.Background())
}

func TestWithMetric_withEndpoint(t *testing.T) {
	t.Parallel()

	sdk, err := o11y.New(
		context.Background(),
		o11y.Config{ServiceName: "svc"},
		otlpgrpc.WithMetric("localhost:4317"),
	)
	if err != nil {
		t.Fatalf("New() with WithMetric(endpoint) error = %v", err)
	}
	_ = sdk.Shutdown(context.Background())
}

func TestWithLog_noEndpoint(t *testing.T) {
	t.Parallel()

	sdk, err := o11y.New(
		context.Background(),
		o11y.Config{ServiceName: "svc"},
		otlpgrpc.WithLog(),
	)
	if err != nil {
		t.Fatalf("New() with WithLog() error = %v", err)
	}
	_ = sdk.Shutdown(context.Background())
}

func TestWithLog_withEndpoint(t *testing.T) {
	t.Parallel()

	sdk, err := o11y.New(
		context.Background(),
		o11y.Config{ServiceName: "svc"},
		otlpgrpc.WithLog("localhost:4317"),
	)
	if err != nil {
		t.Fatalf("New() with WithLog(endpoint) error = %v", err)
	}
	_ = sdk.Shutdown(context.Background())
}
