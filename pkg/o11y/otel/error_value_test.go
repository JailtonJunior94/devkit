package otel

import (
	"bytes"
	"context"
	"testing"

	observability "devkit/pkg/o11y"
	"go.opentelemetry.io/otel/attribute"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

type typedNilError struct {
	message string
}

func (e *typedNilError) Error() string {
	return e.message
}

func TestConvertFieldToAttributeHandlesTypedNilError(t *testing.T) {
	t.Parallel()

	var errValue error = (*typedNilError)(nil)
	attr := convertFieldToAttribute(observability.Error(errValue))

	if attr.Key != attribute.Key("error") {
		t.Fatalf("unexpected attribute key: %q", attr.Key)
	}

	if got := attr.Value.AsString(); got != "<nil>" {
		t.Fatalf("unexpected attribute value: %q", got)
	}
}

func TestLoggerInfoHandlesTypedNilErrorField(t *testing.T) {
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

	var errValue error = (*typedNilError)(nil)
	logger.Info(context.Background(), "hello", observability.Error(errValue))

	if got := output.String(); !bytes.Contains([]byte(got), []byte("error=<nil>")) {
		t.Fatalf("expected typed nil error to render safely, got %q", got)
	}
}
