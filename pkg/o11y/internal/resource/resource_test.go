package resource_test

import (
	"testing"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"

	"devkit/pkg/o11y/internal/resource"
)

func TestBuildIncludesConfiguredAttributes(t *testing.T) {
	t.Parallel()

	res, err := resource.Build(resource.Config{
		ServiceName:    "checkout",
		ServiceVersion: "1.2.3",
		Environment:    "staging",
		Attributes: []attribute.KeyValue{
			attribute.String("custom.key", "custom-value"),
		},
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	set := res.Set()

	assertResourceValue(t, set, semconv.ServiceNameKey, "checkout")
	assertResourceValue(t, set, semconv.ServiceVersionKey, "1.2.3")
	assertResourceValue(t, set, semconv.DeploymentEnvironmentNameKey, "staging")
	assertResourceValue(t, set, attribute.Key("custom.key"), "custom-value")
}

func TestBuildMergesWithDefaultResource(t *testing.T) {
	t.Parallel()

	res, err := resource.Build(resource.Config{ServiceName: "checkout"})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	assertResourceValue(t, res.Set(), semconv.ServiceNameKey, "checkout")
}

func assertResourceValue(t *testing.T, set interface{ Value(attribute.Key) (attribute.Value, bool) }, key attribute.Key, want string) {
	t.Helper()

	got, ok := set.Value(key)
	if !ok {
		t.Fatalf("resource attribute %q not found", key)
	}
	if got.AsString() != want {
		t.Fatalf("resource attribute %q = %q, want %q", key, got.AsString(), want)
	}
}
