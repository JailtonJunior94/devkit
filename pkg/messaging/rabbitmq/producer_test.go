package rabbitmq_test

import (
	"bytes"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"devkit/pkg/messaging"
	"devkit/pkg/messaging/rabbitmq"
)

func TestNewProducer_MissingURI(t *testing.T) {
	_, err := rabbitmq.NewProducer()
	if err == nil {
		t.Fatal("expected error for missing URI, got nil")
	}
	if err != rabbitmq.ErrNoURI {
		t.Errorf("expected ErrNoURI, got %v", err)
	}
}

func TestNewProducer_ValidOptions(t *testing.T) {
	p, err := rabbitmq.NewProducer(
		rabbitmq.WithProducerURI("amqp://localhost"),
		rabbitmq.WithProducerExchange("events"),
		rabbitmq.WithProducerRoutingKey("order.created"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil producer")
	}
}

func TestProducerOptions_Applied(t *testing.T) {
	tests := []struct {
		name string
		opts []rabbitmq.ProducerOption
	}{
		{
			name: "confirm mode",
			opts: []rabbitmq.ProducerOption{
				rabbitmq.WithProducerURI("amqp://localhost"),
				rabbitmq.WithProducerConfirm(),
			},
		},
		{
			name: "max retries override",
			opts: []rabbitmq.ProducerOption{
				rabbitmq.WithProducerURI("amqp://localhost"),
				rabbitmq.WithProducerMaxRetries(5),
			},
		},
		{
			name: "custom backoff",
			opts: []rabbitmq.ProducerOption{
				rabbitmq.WithProducerURI("amqp://localhost"),
				rabbitmq.WithProducerBackoff(200*time.Millisecond, 10*time.Second, 3.0),
			},
		},
		{
			name: "mandatory flag",
			opts: []rabbitmq.ProducerOption{
				rabbitmq.WithProducerURI("amqp://localhost"),
				rabbitmq.WithProducerMandatory(true),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := rabbitmq.NewProducer(tt.opts...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p == nil {
				t.Fatal("expected non-nil producer")
			}
		})
	}
}

func TestSerializeMessage(t *testing.T) {
	tests := []struct {
		name        string
		msg         messaging.Message
		wantBody    []byte
		wantHeaders map[string]string
	}{
		{
			name: "body and headers copied",
			msg: messaging.Message{
				EventType: "order.created",
				Headers:   map[string]string{"x-request-id": "abc"},
				Body:      []byte(`{"id":1}`),
			},
			wantBody:    []byte(`{"id":1}`),
			wantHeaders: map[string]string{"event_type": "order.created", "x-request-id": "abc"},
		},
		{
			name: "empty event type not added to headers",
			msg: messaging.Message{
				Headers: map[string]string{"foo": "bar"},
				Body:    []byte("data"),
			},
			wantBody:    []byte("data"),
			wantHeaders: map[string]string{"foo": "bar"},
		},
		{
			name: "nil headers",
			msg: messaging.Message{
				EventType: "ping",
				Body:      []byte("pong"),
			},
			wantBody:    []byte("pong"),
			wantHeaders: map[string]string{"event_type": "ping"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pub := rabbitmq.SerializeMessage(tt.msg)

			if !bytes.Equal(pub.Body, tt.wantBody) {
				t.Errorf("Body: got %q, want %q", pub.Body, tt.wantBody)
			}

			for k, want := range tt.wantHeaders {
				got, ok := pub.Headers[k]
				if !ok {
					t.Errorf("header %q missing", k)
					continue
				}
				if got != want {
					t.Errorf("header %q: got %v, want %v", k, got, want)
				}
			}
		})
	}
}

func TestPublishConfig_Options(t *testing.T) {
	cfg := messaging.PublishConfig{}

	messaging.WithExchange("my-exchange")(&cfg)
	messaging.WithRoutingKey("my.key")(&cfg)
	messaging.WithMandatory(true)(&cfg)

	if cfg.Exchange != "my-exchange" {
		t.Errorf("Exchange: got %q, want %q", cfg.Exchange, "my-exchange")
	}
	if cfg.RoutingKey != "my.key" {
		t.Errorf("RoutingKey: got %q, want %q", cfg.RoutingKey, "my.key")
	}
	if !cfg.Mandatory {
		t.Error("Mandatory: expected true")
	}
}

// TestSerializeMessage_ContentType verifies the content type is set.
func TestSerializeMessage_ContentType(t *testing.T) {
	pub := rabbitmq.SerializeMessage(messaging.Message{Body: []byte("x")})
	if pub.ContentType != "application/octet-stream" {
		t.Errorf("ContentType: got %q, want %q", pub.ContentType, "application/octet-stream")
	}
}

// Ensure SerializeMessage returns amqp.Publishing — type assertion from export.
var _ amqp.Publishing = rabbitmq.SerializeMessage(messaging.Message{})
