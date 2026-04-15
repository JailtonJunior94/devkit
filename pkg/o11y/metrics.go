package o11y

import (
	"context"
	"errors"
)

// ErrNilGaugeCallback is returned when a nil callback is passed to Gauge.
var ErrNilGaugeCallback = errors.New("gauge callback cannot be nil")

// Metrics exposes application metric instruments.
type Metrics interface {
	Counter(name, description, unit string) (Counter, error)
	Histogram(name, description, unit string) (Histogram, error)
	UpDownCounter(name, description, unit string) (UpDownCounter, error)
	Gauge(name, description, unit string, callback GaugeCallback) error
}

// Counter only accepts positive increments.
type Counter interface {
	Add(ctx context.Context, value int64, fields ...Field)
	Increment(ctx context.Context, fields ...Field)
}

// Histogram records value distributions.
type Histogram interface {
	Record(ctx context.Context, value float64, fields ...Field)
}

// UpDownCounter accepts positive and negative deltas.
type UpDownCounter interface {
	Add(ctx context.Context, value int64, fields ...Field)
}

// GaugeCallback returns the current value for an asynchronous gauge.
type GaugeCallback func(ctx context.Context) float64
