package otel

import (
	"bytes"
	"context"
	"strings"
	"testing"

	observability "devkit/pkg/o11y"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

func TestLoggerWithSanitizesPersistentFields(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	loggerProvider := sdklog.NewLoggerProvider()
	t.Cleanup(func() {
		if err := loggerProvider.Shutdown(context.Background()); err != nil {
			t.Fatalf("shutdown logger provider: %v", err)
		}
	})

	baseLogger := &otelLogger{
		otelLog:     loggerProvider.Logger("test"),
		slogLogger:  createSlogLogger(observability.LogLevelInfo, observability.LogFormatText, &output),
		level:       observability.LogLevelInfo,
		format:      observability.LogFormatText,
		serviceName: "svc",
	}

	childLogger := baseLogger.With(observability.String("authorization", "secret-token"))
	childLogger.Info(nilContext(), "hello", observability.String("api_key", "another-secret"))

	logLine := output.String()
	if strings.Contains(logLine, "secret-token") || strings.Contains(logLine, "another-secret") {
		t.Fatalf("expected sensitive values to be redacted, got %q", logLine)
	}

	if !strings.Contains(logLine, "authorization=[REDACTED]") {
		t.Fatalf("expected persistent field to be redacted, got %q", logLine)
	}

	if !strings.Contains(logLine, "api_key=[REDACTED]") {
		t.Fatalf("expected call field to be redacted, got %q", logLine)
	}
}

func TestLoggerHandlesNilContext(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	loggerProvider := sdklog.NewLoggerProvider()
	t.Cleanup(func() {
		if err := loggerProvider.Shutdown(context.Background()); err != nil {
			t.Fatalf("shutdown logger provider: %v", err)
		}
	})

	logger := &otelLogger{
		otelLog:     loggerProvider.Logger("test"),
		slogLogger:  createSlogLogger(observability.LogLevelInfo, observability.LogFormatText, &output),
		level:       observability.LogLevelInfo,
		format:      observability.LogFormatText,
		serviceName: "svc",
	}

	logger.Info(nilContext(), "hello")

	if !strings.Contains(output.String(), "hello") {
		t.Fatalf("expected log output to contain message, got %q", output.String())
	}
}

func nilContext() context.Context {
	return nil
}
