package otel

import (
	"context"
	"io"
	"log/slog"
	"os"
	"time"

	observability "devkit/pkg/o11y"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/trace"
)

type otelLogger struct {
	otelLog     otellog.Logger
	slogLogger  *slog.Logger
	level       observability.LogLevel
	format      observability.LogFormat
	serviceName string
	fields      []observability.Field
}

func newOtelLogger(
	level observability.LogLevel,
	format observability.LogFormat,
	serviceName string,
	otelLog otellog.Logger,
) *otelLogger {
	return &otelLogger{
		otelLog:     otelLog,
		slogLogger:  createSlogLogger(level, format, os.Stdout),
		level:       level,
		format:      format,
		serviceName: serviceName,
		fields:      make([]observability.Field, 0),
	}
}

func createSlogLogger(level observability.LogLevel, format observability.LogFormat, output io.Writer) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: convertLogLevel(level),
	}

	if format == observability.LogFormatJSON {
		return slog.New(slog.NewJSONHandler(output, opts))
	}

	return slog.New(slog.NewTextHandler(output, opts))
}

func convertLogLevel(level observability.LogLevel) slog.Level {
	switch level {
	case observability.LogLevelDebug:
		return slog.LevelDebug
	case observability.LogLevelInfo:
		return slog.LevelInfo
	case observability.LogLevelWarn:
		return slog.LevelWarn
	case observability.LogLevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func (l *otelLogger) Debug(ctx context.Context, msg string, fields ...observability.Field) {
	l.log(ctx, slog.LevelDebug, msg, fields...)
}

func (l *otelLogger) Info(ctx context.Context, msg string, fields ...observability.Field) {
	l.log(ctx, slog.LevelInfo, msg, fields...)
}

func (l *otelLogger) Warn(ctx context.Context, msg string, fields ...observability.Field) {
	l.log(ctx, slog.LevelWarn, msg, fields...)
}

func (l *otelLogger) Error(ctx context.Context, msg string, fields ...observability.Field) {
	l.log(ctx, slog.LevelError, msg, fields...)
}

func (l *otelLogger) log(ctx context.Context, level slog.Level, msg string, fields ...observability.Field) {
	ctx = normalizeContext(ctx)

	if msg == "" {
		msg = "[empty message]"
	}

	fields = sanitizeFields(fields)

	allFields := make([]observability.Field, 0, len(l.fields)+len(fields)+3)
	allFields = append(allFields, l.fields...)
	allFields = append(allFields, fields...)

	span := trace.SpanFromContext(ctx)
	spanContext := span.SpanContext()
	if spanContext.IsValid() {
		allFields = append(allFields,
			observability.String("trace_id", spanContext.TraceID().String()),
			observability.String("span_id", spanContext.SpanID().String()),
		)
	}

	allFields = append(allFields, observability.String("service", l.serviceName))

	attrs := make([]slog.Attr, 0, len(allFields))
	for _, field := range allFields {
		attrs = append(attrs, convertFieldToSlogAttr(field))
	}

	l.slogLogger.LogAttrs(ctx, level, msg, attrs...)

	l.emitOTLPLog(ctx, level, msg, allFields)
}

func (l *otelLogger) emitOTLPLog(
	ctx context.Context,
	level slog.Level,
	msg string,
	fields []observability.Field,
) {
	attrs := make([]otellog.KeyValue, 0, len(fields))
	for _, field := range fields {
		attrs = append(attrs, convertFieldToOTelAttr(field))
	}

	record := otellog.Record{}
	record.SetTimestamp(time.Now())
	record.SetBody(otellog.StringValue(msg))
	record.SetSeverity(convertSlogLevelToOTel(level))
	record.SetSeverityText(level.String())
	record.AddAttributes(attrs...)

	l.otelLog.Emit(ctx, record)
}

func (l *otelLogger) With(fields ...observability.Field) observability.Logger {
	combinedFields := append(cloneFields(l.fields), fields...)
	combinedFields = sanitizeFields(combinedFields)

	return &otelLogger{
		otelLog:     l.otelLog,
		slogLogger:  l.slogLogger,
		level:       l.level,
		format:      l.format,
		serviceName: l.serviceName,
		fields:      combinedFields,
	}
}
