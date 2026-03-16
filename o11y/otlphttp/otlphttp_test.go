package otlphttp_test

import (
	"context"
	"testing"

	"devkit/o11y"
	"devkit/o11y/otlphttp"
)

func TestWithTrace_noEndpoint(t *testing.T) {
	t.Parallel()

	sdk, err := o11y.New(
		context.Background(),
		o11y.Config{ServiceName: "svc"},
		otlphttp.WithTrace(),
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
		otlphttp.WithTrace("localhost:4318"),
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
		otlphttp.WithMetric(),
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
		otlphttp.WithMetric("localhost:4318"),
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
		otlphttp.WithLog(),
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
		otlphttp.WithLog("localhost:4318"),
	)
	if err != nil {
		t.Fatalf("New() with WithLog(endpoint) error = %v", err)
	}
	_ = sdk.Shutdown(context.Background())
}
