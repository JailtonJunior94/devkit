package messaging

import "context"

// Producer defines the contract for publishing messages to a broker.
// Implementations must be safe for concurrent use.
type Producer interface {
	// Publish sends msg to the broker. Options override per-message routing config.
	Publish(ctx context.Context, msg Message, opts ...PublishOption) error
	// Shutdown closes producer resources best-effort while respecting ctx.
	// Implementations do not guarantee reuse after shutdown.
	Shutdown(ctx context.Context) error
}

// PublishOption configures per-message publish behaviour.
type PublishOption func(*PublishConfig)

// PublishConfig holds routing configuration applied per Publish call.
type PublishConfig struct {
	Exchange   string
	RoutingKey string
	Mandatory  bool
}

// WithExchange sets the target exchange for the publish call.
func WithExchange(exchange string) PublishOption {
	return func(c *PublishConfig) { c.Exchange = exchange }
}

// WithRoutingKey sets the routing key for the publish call.
func WithRoutingKey(key string) PublishOption {
	return func(c *PublishConfig) { c.RoutingKey = key }
}

// WithMandatory sets the mandatory flag for the publish call.
// When true, the broker returns the message if it cannot be routed.
func WithMandatory(m bool) PublishOption {
	return func(c *PublishConfig) { c.Mandatory = m }
}
