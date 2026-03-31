package kafka

import "errors"

var (
	ErrNoBrokers              = errors.New("kafka: at least one broker address is required")
	ErrNoGroupID              = errors.New("kafka: consumer group ID is required")
	ErrNoTopics               = errors.New("kafka: at least one topic is required")
	ErrMultipleAuthMechanisms = errors.New("kafka: only one authentication mechanism may be configured")
	ErrShutdownTimeout        = errors.New("kafka: shutdown timed out waiting for in-flight workers")
)
