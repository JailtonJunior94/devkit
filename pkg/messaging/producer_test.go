package messaging_test

import (
	"testing"

	"devkit/pkg/messaging"
)

func TestPublishOptions(t *testing.T) {
	tests := []struct {
		name     string
		opts     []messaging.PublishOption
		want     messaging.PublishConfig
	}{
		{
			name: "no options yields zero config",
			opts: nil,
			want: messaging.PublishConfig{},
		},
		{
			name: "WithExchange sets exchange",
			opts: []messaging.PublishOption{messaging.WithExchange("events")},
			want: messaging.PublishConfig{Exchange: "events"},
		},
		{
			name: "WithRoutingKey sets routing key",
			opts: []messaging.PublishOption{messaging.WithRoutingKey("order.created")},
			want: messaging.PublishConfig{RoutingKey: "order.created"},
		},
		{
			name: "WithMandatory sets mandatory true",
			opts: []messaging.PublishOption{messaging.WithMandatory(true)},
			want: messaging.PublishConfig{Mandatory: true},
		},
		{
			name: "all options combined",
			opts: []messaging.PublishOption{
				messaging.WithExchange("x"),
				messaging.WithRoutingKey("rk"),
				messaging.WithMandatory(true),
			},
			want: messaging.PublishConfig{Exchange: "x", RoutingKey: "rk", Mandatory: true},
		},
		{
			name: "last option wins for same field",
			opts: []messaging.PublishOption{
				messaging.WithExchange("first"),
				messaging.WithExchange("second"),
			},
			want: messaging.PublishConfig{Exchange: "second"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := messaging.PublishConfig{}
			for _, opt := range tc.opts {
				opt(&cfg)
			}
			if cfg != tc.want {
				t.Errorf("got %+v, want %+v", cfg, tc.want)
			}
		})
	}
}
