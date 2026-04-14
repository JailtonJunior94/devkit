package rabbitmq_test

import (
	"errors"
	"testing"

	"devkit/pkg/messaging/rabbitmq"
)

func TestSentinelErrors_Distinct(t *testing.T) {
	sentinels := []error{
		rabbitmq.ErrNoURI,
		rabbitmq.ErrNoQueues,
		rabbitmq.ErrNoPrefetch,
		rabbitmq.ErrShutdown,
		rabbitmq.ErrConnection,
		rabbitmq.ErrChannel,
	}

	for i, a := range sentinels {
		for j, b := range sentinels {
			if i == j {
				continue
			}
			if errors.Is(a, b) {
				t.Errorf("sentinel[%d] (%v) incorrectly matches sentinel[%d] (%v)", i, a, j, b)
			}
		}
	}
}

func TestSentinelErrors_ErrorsIs(t *testing.T) {
	tests := []struct {
		name     string
		sentinel error
	}{
		{"ErrNoURI", rabbitmq.ErrNoURI},
		{"ErrNoQueues", rabbitmq.ErrNoQueues},
		{"ErrNoPrefetch", rabbitmq.ErrNoPrefetch},
		{"ErrShutdown", rabbitmq.ErrShutdown},
		{"ErrConnection", rabbitmq.ErrConnection},
		{"ErrChannel", rabbitmq.ErrChannel},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wrapped := errors.Join(tc.sentinel)
			if !errors.Is(wrapped, tc.sentinel) {
				t.Errorf("errors.Is failed for %v", tc.sentinel)
			}
		})
	}
}
