package o11y_test

import (
	"reflect"
	"testing"

	"devkit/pkg/o11y"
)

func TestNewSpanConfigAppliesOptions(t *testing.T) {
	t.Parallel()

	config := o11y.NewSpanConfig([]o11y.SpanOption{
		o11y.WithSpanKind(o11y.SpanKindClient),
		o11y.WithAttributes(
			o11y.String("component", "checkout"),
			o11y.Int("attempt", 2),
		),
	})

	if config.Kind() != o11y.SpanKindClient {
		t.Fatalf("expected span kind client, got %v", config.Kind())
	}

	attributes := config.Attributes()
	if len(attributes) != 2 {
		t.Fatalf("expected 2 attributes, got %d", len(attributes))
	}

	if attributes[0] != o11y.String("component", "checkout") {
		t.Fatalf("unexpected first attribute: %#v", attributes[0])
	}

	if attributes[1] != o11y.Int("attempt", 2) {
		t.Fatalf("unexpected second attribute: %#v", attributes[1])
	}
}

func TestFieldConstructors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		field o11y.Field
		key   string
		value any
	}{
		{name: "string", field: o11y.String("service", "payments"), key: "service", value: "payments"},
		{name: "int", field: o11y.Int("attempt", 3), key: "attempt", value: 3},
		{name: "int64", field: o11y.Int64("duration_ms", int64(42)), key: "duration_ms", value: int64(42)},
		{name: "float64", field: o11y.Float64("latency", 1.5), key: "latency", value: 1.5},
		{name: "bool", field: o11y.Bool("sampled", true), key: "sampled", value: true},
		{name: "any", field: o11y.Any("payload", map[string]int{"a": 1}), key: "payload", value: map[string]int{"a": 1}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.field.Key != tt.key {
				t.Fatalf("expected key %q, got %q", tt.key, tt.field.Key)
			}

			if !reflect.DeepEqual(tt.field.Value, tt.value) {
				t.Fatalf("expected value %#v, got %#v", tt.value, tt.field.Value)
			}
		})
	}
}

func TestNewSpanConfigIgnoresNilOptionAndKeepsAttributesImmutable(t *testing.T) {
	t.Parallel()

	config := o11y.NewSpanConfig([]o11y.SpanOption{
		nil,
		o11y.WithAttributes(o11y.String("component", "checkout")),
	})

	attrs := config.Attributes()
	if len(attrs) != 1 {
		t.Fatalf("expected 1 attribute, got %d", len(attrs))
	}

	attrs[0] = o11y.String("component", "mutated")

	fresh := config.Attributes()
	if fresh[0] != o11y.String("component", "checkout") {
		t.Fatalf("expected config attributes to remain immutable, got %#v", fresh)
	}
}
