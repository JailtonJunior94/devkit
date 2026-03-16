// Package otlphttp provides o11y options that configure OTLP HTTP exporters
// for traces, metrics, and logs. Import this package only when you need OTLP
// HTTP export; doing so pulls in the HTTP and protobuf transitive dependencies.
package otlphttp

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"

	"devkit/o11y"
)

// WithTrace returns an o11y.Option that configures an OTLP HTTP span exporter.
// The optional endpoint overrides the default OTLP HTTP endpoint
// (OTEL_EXPORTER_OTLP_ENDPOINT or localhost:4318).
func WithTrace(endpoint ...string) o11y.Option {
	return func(ctx context.Context, cfg *o11y.Config) error {
		var opts []otlptracehttp.Option
		if len(endpoint) > 0 && endpoint[0] != "" {
			opts = append(opts, otlptracehttp.WithEndpoint(endpoint[0]))
		}
		exp, err := otlptracehttp.New(ctx, opts...)
		if err != nil {
			return fmt.Errorf("otlphttp: creating trace exporter: %w", err)
		}
		cfg.TraceExporter = exp
		return nil
	}
}

// WithMetric returns an o11y.Option that configures an OTLP HTTP metric exporter.
// The optional endpoint overrides the default OTLP HTTP endpoint
// (OTEL_EXPORTER_OTLP_ENDPOINT or localhost:4318).
func WithMetric(endpoint ...string) o11y.Option {
	return func(ctx context.Context, cfg *o11y.Config) error {
		var opts []otlpmetrichttp.Option
		if len(endpoint) > 0 && endpoint[0] != "" {
			opts = append(opts, otlpmetrichttp.WithEndpoint(endpoint[0]))
		}
		exp, err := otlpmetrichttp.New(ctx, opts...)
		if err != nil {
			return fmt.Errorf("otlphttp: creating metric exporter: %w", err)
		}
		cfg.MetricExporter = exp
		return nil
	}
}

// WithLog returns an o11y.Option that configures an OTLP HTTP log exporter.
// The optional endpoint overrides the default OTLP HTTP endpoint
// (OTEL_EXPORTER_OTLP_ENDPOINT or localhost:4318).
func WithLog(endpoint ...string) o11y.Option {
	return func(ctx context.Context, cfg *o11y.Config) error {
		var opts []otlploghttp.Option
		if len(endpoint) > 0 && endpoint[0] != "" {
			opts = append(opts, otlploghttp.WithEndpoint(endpoint[0]))
		}
		exp, err := otlploghttp.New(ctx, opts...)
		if err != nil {
			return fmt.Errorf("otlphttp: creating log exporter: %w", err)
		}
		cfg.LogExporter = exp
		return nil
	}
}
