// Package otlpgrpc provides optional OTLP gRPC options for the tracing module.
package otlpgrpc

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"

	"devkit/pkg/o11y/tracing"
)

// WithOTLPGRPC creates an OTLP gRPC span exporter for the tracing module.
// The optional endpoint overrides OTEL_EXPORTER_OTLP_ENDPOINT or localhost:4317.
func WithOTLPGRPC(endpoint ...string) tracing.Option {
	return func(ctx context.Context, cfg *tracing.Config) error {
		var opts []otlptracegrpc.Option
		if len(endpoint) > 0 && endpoint[0] != "" {
			opts = append(opts, otlptracegrpc.WithEndpoint(endpoint[0]))
		}
		exp, err := otlptracegrpc.New(ctx, opts...)
		if err != nil {
			return fmt.Errorf("tracing/otlpgrpc: creating trace exporter: %w", err)
		}
		cfg.SpanExporter = exp
		return nil
	}
}
