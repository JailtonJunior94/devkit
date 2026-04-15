// Package otlpgrpc provides optional OTLP gRPC options for the metrics module.
package otlpgrpc

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"

	"devkit/pkg/o11y/metrics"
)

// WithOTLPGRPC creates an OTLP gRPC metric exporter for the metrics module.
func WithOTLPGRPC(endpoint ...string) metrics.Option {
	return func(ctx context.Context, cfg *metrics.Config) error {
		var opts []otlpmetricgrpc.Option
		if len(endpoint) > 0 && endpoint[0] != "" {
			opts = append(opts, otlpmetricgrpc.WithEndpoint(endpoint[0]))
		}
		exp, err := otlpmetricgrpc.New(ctx, opts...)
		if err != nil {
			return fmt.Errorf("metrics/otlpgrpc: creating metric exporter: %w", err)
		}
		cfg.Exporter = exp
		return nil
	}
}
