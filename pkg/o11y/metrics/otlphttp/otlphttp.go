// Package otlphttp provides optional OTLP HTTP options for the metrics module.
package otlphttp

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"

	"devkit/pkg/o11y/metrics"
)

// WithOTLPHTTP creates an OTLP HTTP metric exporter for the metrics module.
func WithOTLPHTTP(endpoint ...string) metrics.Option {
	return func(ctx context.Context, cfg *metrics.Config) error {
		var opts []otlpmetrichttp.Option
		if len(endpoint) > 0 && endpoint[0] != "" {
			opts = append(opts, otlpmetrichttp.WithEndpoint(endpoint[0]))
		}
		exp, err := otlpmetrichttp.New(ctx, opts...)
		if err != nil {
			return fmt.Errorf("metrics/otlphttp: creating metric exporter: %w", err)
		}
		cfg.Exporter = exp
		return nil
	}
}
