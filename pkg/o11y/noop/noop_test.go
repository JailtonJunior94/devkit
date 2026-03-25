package noop_test

import (
	"context"
	"testing"

	"devkit/pkg/o11y/noop"
)

func TestNewTracerProvider(t *testing.T) {
	t.Parallel()

	tp := noop.NewTracerProvider()
	if tp == nil {
		t.Fatal("NewTracerProvider() returned nil")
	}

	_, span := tp.Tracer("test").Start(context.Background(), "op")
	span.End()
}

func TestNewMeterProvider(t *testing.T) {
	t.Parallel()

	mp := noop.NewMeterProvider()
	if mp == nil {
		t.Fatal("NewMeterProvider() returned nil")
	}

	counter, err := mp.Meter("test").Int64Counter("ops")
	if err != nil {
		t.Fatalf("Int64Counter() error = %v", err)
	}
	counter.Add(context.Background(), 1)
}

func TestNewLogger(t *testing.T) {
	t.Parallel()

	logger := noop.NewLogger()
	if logger == nil {
		t.Fatal("NewLogger() returned nil")
	}

	logger.Info("discarded")
}
