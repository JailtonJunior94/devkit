// Package otlphttp provides optional OTLP HTTP options for the logging module.
package otlphttp

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"

	"devkit/pkg/o11y/logging"
)

// WithOTLPHTTP creates an OTLP HTTP log exporter for the logging module.
// The optional endpoint overrides OTEL_EXPORTER_OTLP_ENDPOINT or localhost:4318.
func WithOTLPHTTP(endpoint ...string) logging.Option {
	return func(ctx context.Context, cfg *logging.Config) error {
		var opts []otlploghttp.Option
		if len(endpoint) > 0 && endpoint[0] != "" {
			opts = append(opts, otlploghttp.WithEndpoint(endpoint[0]))
		}
		exp, err := otlploghttp.New(ctx, opts...)
		if err != nil {
			return fmt.Errorf("logging/otlphttp: creating log exporter: %w", err)
		}
		cfg.LogExporter = exp
		return nil
	}
}
