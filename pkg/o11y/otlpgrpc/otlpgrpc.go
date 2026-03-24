// Package otlpgrpc provides o11y options that configure OTLP gRPC exporters
// for traces, metrics, and logs. Import this package only when you need OTLP
// gRPC export; doing so pulls in the gRPC and protobuf transitive dependencies.
package otlpgrpc

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"

	"devkit/pkg/o11y"
)

// WithTrace returns an o11y.Option that configures an OTLP gRPC span exporter.
// The optional endpoint overrides the default OTLP gRPC endpoint
// (OTEL_EXPORTER_OTLP_ENDPOINT or localhost:4317).
func WithTrace(endpoint ...string) o11y.Option {
	return func(ctx context.Context, cfg *o11y.Config) error {
		var opts []otlptracegrpc.Option
		if len(endpoint) > 0 && endpoint[0] != "" {
			opts = append(opts, otlptracegrpc.WithEndpoint(endpoint[0]))
		}
		exp, err := otlptracegrpc.New(ctx, opts...)
		if err != nil {
			return fmt.Errorf("otlpgrpc: creating trace exporter: %w", err)
		}
		cfg.TraceExporter = exp
		return nil
	}
}

// WithMetric returns an o11y.Option that configures an OTLP gRPC metric exporter.
// The optional endpoint overrides the default OTLP gRPC endpoint
// (OTEL_EXPORTER_OTLP_ENDPOINT or localhost:4317).
func WithMetric(endpoint ...string) o11y.Option {
	return func(ctx context.Context, cfg *o11y.Config) error {
		var opts []otlpmetricgrpc.Option
		if len(endpoint) > 0 && endpoint[0] != "" {
			opts = append(opts, otlpmetricgrpc.WithEndpoint(endpoint[0]))
		}
		exp, err := otlpmetricgrpc.New(ctx, opts...)
		if err != nil {
			return fmt.Errorf("otlpgrpc: creating metric exporter: %w", err)
		}
		cfg.MetricExporter = exp
		return nil
	}
}

// WithLog returns an o11y.Option that configures an OTLP gRPC log exporter.
// The optional endpoint overrides the default OTLP gRPC endpoint
// (OTEL_EXPORTER_OTLP_ENDPOINT or localhost:4317).
func WithLog(endpoint ...string) o11y.Option {
	return func(ctx context.Context, cfg *o11y.Config) error {
		var opts []otlploggrpc.Option
		if len(endpoint) > 0 && endpoint[0] != "" {
			opts = append(opts, otlploggrpc.WithEndpoint(endpoint[0]))
		}
		exp, err := otlploggrpc.New(ctx, opts...)
		if err != nil {
			return fmt.Errorf("otlpgrpc: creating log exporter: %w", err)
		}
		cfg.LogExporter = exp
		return nil
	}
}
