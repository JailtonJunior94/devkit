package otel

import (
	"fmt"

	observability "devkit/pkg/o11y"
	"go.opentelemetry.io/otel/attribute"
)

func convertFieldToAttribute(field observability.Field) attribute.KeyValue {
	switch v := field.Value.(type) {
	case string:
		return attribute.String(field.Key, v)
	case int:
		return attribute.Int(field.Key, v)
	case int64:
		return attribute.Int64(field.Key, v)
	case float64:
		return attribute.Float64(field.Key, v)
	case bool:
		return attribute.Bool(field.Key, v)
	case error:
		return attribute.String(field.Key, safeErrorString(v))
	default:
		return attribute.String(field.Key, fmt.Sprintf("%v", v))
	}
}

func convertFieldsToAttributes(fields []observability.Field) []attribute.KeyValue {
	if len(fields) == 0 {
		return nil
	}

	attrs := make([]attribute.KeyValue, len(fields))
	for i, field := range fields {
		attrs[i] = convertFieldToAttribute(field)
	}
	return attrs
}
