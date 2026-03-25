// Package otlpgrpc provides optional OTLP gRPC options for the logging module.
package otlpgrpc

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"

	"devkit/pkg/o11y/logging"
)

// WithOTLPGRPC creates an OTLP gRPC log exporter for the logging module.
// The optional endpoint overrides OTEL_EXPORTER_OTLP_ENDPOINT or localhost:4317.
func WithOTLPGRPC(endpoint ...string) logging.Option {
	return func(ctx context.Context, cfg *logging.Config) error {
		var opts []otlploggrpc.Option
		if len(endpoint) > 0 && endpoint[0] != "" {
			opts = append(opts, otlploggrpc.WithEndpoint(endpoint[0]))
		}
		exp, err := otlploggrpc.New(ctx, opts...)
		if err != nil {
			return fmt.Errorf("logging/otlpgrpc: creating log exporter: %w", err)
		}
		cfg.LogExporter = exp
		return nil
	}
}
