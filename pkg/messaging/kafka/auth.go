package kafka

import (
	"crypto/tls"
	"fmt"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
)

func buildDialer(cfg authConfig) (*kafkago.Dialer, error) {
	base := &kafkago.Dialer{
		Timeout:   10 * time.Second,
		DualStack: true,
	}

	switch cfg.mechanism {
	case authNone:
		return base, nil

	case authPlain:
		base.SASLMechanism = plain.Mechanism{
			Username: cfg.username,
			Password: cfg.password,
		}
		base.TLS = &tls.Config{MinVersion: tls.VersionTLS12}
		return base, nil

	case authSCRAM256:
		m, err := scram.Mechanism(scram.SHA256, cfg.username, cfg.password)
		if err != nil {
			return nil, fmt.Errorf("kafka: failed to build SCRAM-SHA-256 mechanism: %w", err)
		}
		base.SASLMechanism = m
		base.TLS = &tls.Config{MinVersion: tls.VersionTLS12}
		return base, nil

	case authSCRAM512:
		m, err := scram.Mechanism(scram.SHA512, cfg.username, cfg.password)
		if err != nil {
			return nil, fmt.Errorf("kafka: failed to build SCRAM-SHA-512 mechanism: %w", err)
		}
		base.SASLMechanism = m
		base.TLS = &tls.Config{MinVersion: tls.VersionTLS12}
		return base, nil

	case authTLS:
		tlsCfg := cfg.tlsCfg
		if tlsCfg == nil {
			tlsCfg = &tls.Config{MinVersion: tls.VersionTLS12}
		}
		base.TLS = tlsCfg
		return base, nil
	}

	return base, nil
}
