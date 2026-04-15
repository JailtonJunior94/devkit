package otel

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	observability "devkit/pkg/o11y"
	otellog "go.opentelemetry.io/otel/log"
)

const (
	redactedValue       = "[REDACTED]"
	maxFields           = 50
	maxFieldValueLength = 2048
)

var defaultSensitiveKeys = []string{
	"password", "passwd", "pwd", "secret", "token", "api_key", "apikey", "api-key",
	"authorization", "auth", "credential", "credentials", "private_key", "privatekey",
	"ssn", "social_security", "credit_card", "creditcard", "card_number", "cvv", "pin",
	"access_token", "refresh_token", "bearer", "session", "cookie",
}

var sensitiveKeysLower = initSensitiveKeysLower()

func initSensitiveKeysLower() []string {
	lower := make([]string, len(defaultSensitiveKeys))
	for i, k := range defaultSensitiveKeys {
		lower[i] = strings.ToLower(k)
	}
	return lower
}

func convertSlogLevelToOTel(level slog.Level) otellog.Severity {
	switch level {
	case slog.LevelDebug:
		return otellog.SeverityDebug
	case slog.LevelInfo:
		return otellog.SeverityInfo
	case slog.LevelWarn:
		return otellog.SeverityWarn
	case slog.LevelError:
		return otellog.SeverityError
	default:
		return otellog.SeverityInfo
	}
}

func convertFieldToOTelAttr(field observability.Field) otellog.KeyValue {
	switch v := field.Value.(type) {
	case string:
		return otellog.String(field.Key, v)
	case int:
		return otellog.Int(field.Key, v)
	case int64:
		return otellog.Int64(field.Key, v)
	case float64:
		return otellog.Float64(field.Key, v)
	case bool:
		return otellog.Bool(field.Key, v)
	case error:
		return otellog.String(field.Key, safeErrorString(v))
	default:
		return otellog.String(field.Key, fmt.Sprint(v))
	}
}

func convertFieldToSlogAttr(field observability.Field) slog.Attr {
	switch v := field.Value.(type) {
	case string:
		return slog.String(field.Key, v)
	case int:
		return slog.Int(field.Key, v)
	case int64:
		return slog.Int64(field.Key, v)
	case float64:
		return slog.Float64(field.Key, v)
	case bool:
		return slog.Bool(field.Key, v)
	case error:
		return slog.String(field.Key, safeErrorString(v))
	default:
		return slog.Any(field.Key, v)
	}
}

func sanitizeFields(fields []observability.Field) []observability.Field {
	if len(fields) > maxFields {
		fields = fields[:maxFields]
	}

	needsSanitization := false
	for _, field := range fields {
		if isSensitiveKey(field.Key) {
			needsSanitization = true
			break
		}
		if s, ok := field.Value.(string); ok && len(s) > maxFieldValueLength {
			needsSanitization = true
			break
		}
	}

	if !needsSanitization {
		return fields
	}

	sanitized := make([]observability.Field, len(fields))
	for i, field := range fields {
		if isSensitiveKey(field.Key) {
			sanitized[i] = observability.String(field.Key, redactedValue)
			continue
		}

		if s, ok := field.Value.(string); ok {
			if len(s) > maxFieldValueLength {
				sanitized[i] = observability.String(field.Key, s[:maxFieldValueLength]+"...[truncated]")
				continue
			}
		}

		sanitized[i] = field
	}

	return sanitized
}

func isSensitiveKey(key string) bool {
	keyLower := strings.ToLower(key)
	for _, sensitive := range sensitiveKeysLower {
		if strings.Contains(keyLower, sensitive) {
			return true
		}
	}
	return false
}

func cloneFields(fields []observability.Field) []observability.Field {
	if len(fields) == 0 {
		return nil
	}

	cloned := make([]observability.Field, len(fields))
	copy(cloned, fields)
	return cloned
}

func normalizeContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}

	return ctx
}
