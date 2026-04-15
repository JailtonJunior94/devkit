// Package resource builds an OpenTelemetry Resource from service identity fields.
package resource

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

// Config holds the minimal fields needed to build an OTel Resource.
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	Attributes     []attribute.KeyValue
}

// Build creates a Resource by merging service identity attributes with the default resource.
func Build(cfg Config) (*resource.Resource, error) {
	attrs := []attribute.KeyValue{
		semconv.ServiceName(cfg.ServiceName),
	}
	if cfg.ServiceVersion != "" {
		attrs = append(attrs, semconv.ServiceVersion(cfg.ServiceVersion))
	}
	if cfg.Environment != "" {
		attrs = append(attrs, semconv.DeploymentEnvironmentName(cfg.Environment))
	}
	attrs = append(attrs, cfg.Attributes...)

	res := resource.NewWithAttributes(semconv.SchemaURL, attrs...)
	return resource.Merge(resource.Default(), res)
}
