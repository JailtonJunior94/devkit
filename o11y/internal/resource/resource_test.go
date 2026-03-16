package resource_test

import (
	"testing"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"

	"devkit/o11y/internal/resource"
)

func TestBuild(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		cfg        resource.Config
		wantKey    attribute.Key
		wantVal    string
		wantNoKey  *attribute.Key
	}{
		{
			name:    "service name only",
			cfg:     resource.Config{ServiceName: "svc"},
			wantKey: semconv.ServiceNameKey,
			wantVal: "svc",
		},
		{
			name: "with version",
			cfg:  resource.Config{ServiceName: "svc", ServiceVersion: "1.0.0"},
			wantKey: semconv.ServiceVersionKey,
			wantVal: "1.0.0",
		},
		{
			name: "with environment",
			cfg:  resource.Config{ServiceName: "svc", Environment: "production"},
			wantKey: semconv.DeploymentEnvironmentNameKey,
			wantVal: "production",
		},
		{
			name: "with extra attributes",
			cfg: resource.Config{
				ServiceName: "svc",
				Attributes:  []attribute.KeyValue{attribute.String("custom.key", "custom-value")},
			},
			wantKey: attribute.Key("custom.key"),
			wantVal: "custom-value",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res, err := resource.Build(tc.cfg)
			if err != nil {
				t.Fatalf("Build() error = %v", err)
			}
			if res == nil {
				t.Fatal("Build() returned nil resource")
			}

			set := res.Set()
			val, ok := set.Value(tc.wantKey)
			if !ok {
				t.Errorf("attribute %q not found in resource", tc.wantKey)
			} else if val.AsString() != tc.wantVal {
				t.Errorf("attribute %q = %q, want %q", tc.wantKey, val.AsString(), tc.wantVal)
			}
		})
	}
}

func TestBuildMergesDefault(t *testing.T) {
	t.Parallel()

	res, err := resource.Build(resource.Config{ServiceName: "svc"})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	// Default resource must include service.name we provided.
	set := res.Set()
	val, ok := set.Value(semconv.ServiceNameKey)
	if !ok {
		t.Fatal("service.name not found after merge with Default")
	}
	if val.AsString() != "svc" {
		t.Errorf("service.name = %q, want %q", val.AsString(), "svc")
	}
}

func BenchmarkBuild(b *testing.B) {
	cfg := resource.Config{
		ServiceName:    "bench-svc",
		ServiceVersion: "1.0.0",
		Environment:    "test",
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = resource.Build(cfg)
	}
}
