package rabbitmq

import (
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"devkit/pkg/messaging"
)

// CalculateBackoff exposes calculateBackoff for testing.
func CalculateBackoff(init, max time.Duration, mul float64, attempt int) time.Duration {
	return calculateBackoff(init, max, mul, attempt)
}

// NewWorkerPoolForTest exposes newWorkerPool for testing.
func NewWorkerPoolForTest(size int) *workerPool {
	return newWorkerPool(size)
}

// NormalizeMessage exposes normalizeMessage for testing.
func NormalizeMessage(d amqp.Delivery, queue, eventTypeHeader string) messaging.Message {
	return normalizeMessage(d, queue, eventTypeHeader)
}

// RetryCountFromHeaders exposes retryCountFromHeaders for testing.
func RetryCountFromHeaders(headers amqp.Table) int {
	return retryCountFromHeaders(headers)
}

// SerializeMessage exposes serializeMessage for testing.
func SerializeMessage(msg messaging.Message) amqp.Publishing {
	return serializeMessage(msg)
}
