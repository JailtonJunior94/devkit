package rabbitmq

// Exchange represents an AMQP exchange declaration.
type Exchange struct {
	Name       string
	Kind       string // "direct", "topic", "fanout", "headers"
	Durable    bool
	AutoDelete bool
	Arguments  map[string]any
}

// Binding represents an AMQP queue-to-exchange binding.
type Binding struct {
	Queue      string
	Exchange   string
	RoutingKey string
	Arguments  map[string]any
}

// QueueDecl holds parameters for a queue declaration.
type QueueDecl struct {
	Name       string
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	Arguments  map[string]any
}
