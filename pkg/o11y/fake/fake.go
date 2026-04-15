// Package fake records observability signals in memory for tests.
package fake

import (
	"context"
	"reflect"

	observability "devkit/pkg/o11y"
)

// Provider records observability signals in memory for tests.
type Provider struct {
	tracer  *FakeTracer
	logger  *FakeLogger
	metrics *FakeMetrics
}

// NewProvider creates an in-memory implementation of the simplified o11y contracts.
func NewProvider() *Provider {
	return &Provider{
		tracer:  NewFakeTracer(),
		logger:  NewFakeLogger(),
		metrics: NewFakeMetrics(),
	}
}

// Tracer returns the in-memory tracer implementation.
func (p *Provider) Tracer() observability.Tracer {
	return p.tracer
}

// Logger returns the in-memory logger implementation.
func (p *Provider) Logger() observability.Logger {
	return p.logger
}

// Metrics returns the in-memory metrics implementation.
func (p *Provider) Metrics() observability.Metrics {
	return p.metrics
}

func ensureContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}

	return ctx
}

// mergeFields creates a new slice combining persistent and call-site fields,
// preventing backing-array corruption when l.fields has spare capacity.
func mergeFields(persistent, callSite []observability.Field) []observability.Field {
	merged := make([]observability.Field, 0, len(persistent)+len(callSite))
	merged = append(merged, persistent...)
	merged = append(merged, callSite...)
	return merged
}

func cloneFakeFields(fields []observability.Field) []observability.Field {
	if len(fields) == 0 {
		return nil
	}

	cloned := make([]observability.Field, len(fields))
	copy(cloned, fields)
	return cloned
}

func isNilError(err error) bool {
	if err == nil {
		return true
	}

	rv := reflect.ValueOf(err)
	if rv.Kind() != reflect.Pointer {
		return false
	}

	return rv.IsNil()
}
