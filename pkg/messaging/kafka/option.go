package kafka

import (
	"crypto/tls"
	"log/slog"
	"time"
)

type Option func(*Consumer)

func WithBrokers(brokers ...string) Option {
	return func(c *Consumer) {
		c.brokers = append(c.brokers, brokers...)
	}
}

func WithGroupID(groupID string) Option {
	return func(c *Consumer) {
		c.groupID = groupID
	}
}

func WithTopics(topics ...string) Option {
	return func(c *Consumer) {
		c.topics = append(c.topics, topics...)
	}
}

func WithWorkers(topic string, n int) Option {
	return func(c *Consumer) {
		if n < 1 {
			n = 1
		}
		c.topicWorkers[topic] = n
		c.topicOrdered[topic] = false
	}
}

func WithOrderedProcessing(topic string) Option {
	return func(c *Consumer) {
		c.topicOrdered[topic] = true
		if _, ok := c.topicWorkers[topic]; !ok {
			c.topicWorkers[topic] = 1
		}
	}
}

func WithMaxRetries(n int) Option {
	return func(c *Consumer) {
		c.maxRetries = n
	}
}

func WithBackoff(initial, max time.Duration, multiplier float64) Option {
	return func(c *Consumer) {
		c.backoffInit = initial
		c.backoffMax = max
		c.backoffMul = multiplier
	}
}

func WithDLQ(enabled bool) Option {
	return func(c *Consumer) {
		c.dlqEnabled = enabled
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(c *Consumer) {
		if logger != nil {
			c.logger = logger
		}
	}
}

func WithEventTypeHeader(key string) Option {
	return func(c *Consumer) {
		if key != "" {
			c.eventTypeHeader = key
		}
	}
}

type authMechanism int

const (
	authNone authMechanism = iota
	authPlain
	authSCRAM256
	authSCRAM512
	authTLS
)

type authConfig struct {
	mechanism authMechanism
	username  string
	password  string
	tlsCfg    *tls.Config
}

func WithPlainAuth(username, password string) Option {
	return func(c *Consumer) {
		c.auth.mechanism = authPlain
		c.auth.username = username
		c.auth.password = password
	}
}

func WithSCRAM256(username, password string) Option {
	return func(c *Consumer) {
		c.auth.mechanism = authSCRAM256
		c.auth.username = username
		c.auth.password = password
	}
}

func WithSCRAM512(username, password string) Option {
	return func(c *Consumer) {
		c.auth.mechanism = authSCRAM512
		c.auth.username = username
		c.auth.password = password
	}
}

func WithTLS(cfg *tls.Config) Option {
	return func(c *Consumer) {
		c.auth.mechanism = authTLS
		c.auth.tlsCfg = cfg
	}
}
