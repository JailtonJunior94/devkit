package otel

import (
	"context"
	"errors"
	"testing"
)

func TestNewProviderReturnsErrorWhenConfigIsNil(t *testing.T) {
	t.Parallel()

	_, err := NewProvider(context.Background(), nil)
	if !errors.Is(err, ErrConfigRequired) {
		t.Fatalf("NewProvider() error = %v, want ErrConfigRequired", err)
	}
}

func TestNewProviderReturnsErrorWhenServiceNameIsEmpty(t *testing.T) {
	t.Parallel()

	_, err := NewProvider(context.Background(), &Config{})
	if !errors.Is(err, ErrServiceNameRequired) {
		t.Fatalf("NewProvider() error = %v, want ErrServiceNameRequired", err)
	}
}

func TestNewProviderCreatesNoopProvidersWhenExportersAreNil(t *testing.T) {
	t.Parallel()

	provider, err := NewProvider(context.Background(), &Config{
		ServiceName: "svc",
	})
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	if provider.Tracer() == nil {
		t.Fatal("Tracer() returned nil")
	}
	if provider.Logger() == nil {
		t.Fatal("Logger() returned nil")
	}
	if provider.Metrics() == nil {
		t.Fatal("Metrics() returned nil")
	}

	if err := provider.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if err := provider.Shutdown(context.Background()); err != nil {
		t.Fatalf("second Shutdown() should be idempotent, got %v", err)
	}
}

func TestDefaultConfigReturnsValidConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig("checkout")
	if cfg.ServiceName != "checkout" {
		t.Fatalf("ServiceName = %q, want %q", cfg.ServiceName, "checkout")
	}

	_, err := NewProvider(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewProvider(DefaultConfig) error = %v", err)
	}
}

