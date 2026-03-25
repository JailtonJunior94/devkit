// Package otlphttp provides optional OTLP HTTP options for the tracing module.
package otlphttp

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"

	"devkit/pkg/o11y/tracing"
)

// WithOTLPHTTP creates an OTLP HTTP span exporter for the tracing module.
// The optional endpoint overrides OTEL_EXPORTER_OTLP_ENDPOINT or localhost:4318.
func WithOTLPHTTP(endpoint ...string) tracing.Option {
	return func(ctx context.Context, cfg *tracing.Config) error {
		var opts []otlptracehttp.Option
		if len(endpoint) > 0 && endpoint[0] != "" {
			opts = append(opts, otlptracehttp.WithEndpoint(endpoint[0]))
		}
		exp, err := otlptracehttp.New(ctx, opts...)
		if err != nil {
			return fmt.Errorf("tracing/otlphttp: creating trace exporter: %w", err)
		}
		cfg.SpanExporter = exp
		return nil
	}
}
