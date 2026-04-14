package rabbitmq

import (
	"testing"
	"time"
)

func TestCalculateBackoff(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		init    time.Duration
		max     time.Duration
		mul     float64
		attempt int
		want    time.Duration
	}{
		{
			name:    "attempt 0 returns init",
			init:    time.Second,
			max:     30 * time.Second,
			mul:     2.0,
			attempt: 0,
			want:    time.Second,
		},
		{
			name:    "attempt 1 doubles",
			init:    time.Second,
			max:     30 * time.Second,
			mul:     2.0,
			attempt: 1,
			want:    2 * time.Second,
		},
		{
			name:    "attempt 2 quadruples",
			init:    time.Second,
			max:     30 * time.Second,
			mul:     2.0,
			attempt: 2,
			want:    4 * time.Second,
		},
		{
			name:    "capped at max",
			init:    time.Second,
			max:     30 * time.Second,
			mul:     2.0,
			attempt: 10,
			want:    30 * time.Second,
		},
		{
			name:    "exactly max boundary",
			init:    time.Second,
			max:     4 * time.Second,
			mul:     2.0,
			attempt: 2,
			want:    4 * time.Second,
		},
		{
			name:    "multiplier 1.5",
			init:    2 * time.Second,
			max:     60 * time.Second,
			mul:     1.5,
			attempt: 3,
			want:    time.Duration(float64(2*time.Second) * 1.5 * 1.5 * 1.5),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := CalculateBackoff(tc.init, tc.max, tc.mul, tc.attempt)
			if got != tc.want {
				t.Errorf("CalculateBackoff(%v, %v, %v, %d) = %v, want %v",
					tc.init, tc.max, tc.mul, tc.attempt, got, tc.want)
			}
		})
	}
}

func TestNewConnectionManager(t *testing.T) {
	t.Parallel()

	mgr := newConnectionManager("amqp://localhost:5672", nil, nil)

	if mgr.uri != "amqp://localhost:5672" {
		t.Errorf("unexpected uri: %s", mgr.uri)
	}
	if mgr.backoffInit != time.Second {
		t.Errorf("unexpected backoffInit: %v", mgr.backoffInit)
	}
	if mgr.backoffMax != 30*time.Second {
		t.Errorf("unexpected backoffMax: %v", mgr.backoffMax)
	}
	if mgr.backoffMul != 2.0 {
		t.Errorf("unexpected backoffMul: %v", mgr.backoffMul)
	}
	if mgr.done == nil {
		t.Error("done channel should be initialized")
	}
}

func TestConnectionManager_ChannelWhenNotConnected(t *testing.T) {
	t.Parallel()

	mgr := newConnectionManager("amqp://localhost:5672", nil, nil)
	_, err := mgr.channel()
	if err == nil {
		t.Fatal("expected error when not connected, got nil")
	}
}
