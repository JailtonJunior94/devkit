package rabbitmq

import "errors"

var (
	// ErrNoURI is returned when no AMQP URI is provided.
	ErrNoURI = errors.New("rabbitmq: AMQP URI is required")

	// ErrNoQueues is returned when no queues are configured for the consumer.
	ErrNoQueues = errors.New("rabbitmq: at least one queue is required")

	// ErrNoPrefetch is returned when prefetch count is not explicitly configured.
	ErrNoPrefetch = errors.New("rabbitmq: prefetch count must be explicitly configured")

	// ErrShutdown is returned when an operation is attempted after shutdown.
	ErrShutdown = errors.New("rabbitmq: consumer is shut down")

	// ErrConnection is returned when the AMQP connection cannot be established or is lost.
	ErrConnection = errors.New("rabbitmq: connection error")

	// ErrChannel is returned when an AMQP channel cannot be created or used.
	ErrChannel = errors.New("rabbitmq: channel error")
)
