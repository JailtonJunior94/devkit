//nolint:unused // fields and methods used by consumer/producer (tasks 5-7)
package rabbitmq

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type connectionManager struct {
	uri    string
	logger *slog.Logger
	tlsCfg *tls.Config

	mu      sync.RWMutex
	conn    *amqp.Connection
	ch      *amqp.Channel
	closeCh chan *amqp.Error

	reconnecting atomic.Bool

	backoffInit time.Duration
	backoffMax  time.Duration
	backoffMul  float64

	onReconnect func() error

	done     chan struct{}
	stopOnce sync.Once
}

func newConnectionManager(uri string, tlsCfg *tls.Config, logger *slog.Logger) *connectionManager {
	if logger == nil {
		logger = slog.Default()
	}
	return &connectionManager{
		uri:         uri,
		logger:      logger,
		tlsCfg:      tlsCfg,
		backoffInit: time.Second,
		backoffMax:  30 * time.Second,
		backoffMul:  2.0,
		done:        make(chan struct{}),
	}
}

func (m *connectionManager) connect() error {
	var conn *amqp.Connection
	var err error

	if m.tlsCfg != nil {
		conn, err = amqp.DialTLS(m.uri, m.tlsCfg)
	} else {
		conn, err = amqp.Dial(m.uri)
	}
	if err != nil {
		return fmt.Errorf("%w: %w", ErrConnection, err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("%w: %w", ErrChannel, err)
	}

	closeCh := make(chan *amqp.Error, 1)
	conn.NotifyClose(closeCh)

	m.mu.Lock()
	m.conn = conn
	m.ch = ch
	m.closeCh = closeCh
	m.mu.Unlock()

	return nil
}

// dial establishes the initial connection. Returns error if connection fails.
func (m *connectionManager) dial() error {
	if err := m.connect(); err != nil {
		return err
	}
	go m.watchConnection()
	return nil
}

func (m *connectionManager) watchConnection() {
	for {
		m.mu.RLock()
		closeCh := m.closeCh
		m.mu.RUnlock()

		select {
		case <-m.done:
			return
		case amqpErr, ok := <-closeCh:
			if !ok {
				// channel closed cleanly (manager is shutting down)
				select {
				case <-m.done:
					return
				default:
				}
			}
			if amqpErr != nil {
				m.logger.Warn("amqp connection lost", "error", amqpErr)
			}
			m.reconnect()
		}
	}
}

func (m *connectionManager) reconnect() {
	if !m.reconnecting.CompareAndSwap(false, true) {
		return
	}
	defer m.reconnecting.Store(false)

	for attempt := 0; ; attempt++ {
		select {
		case <-m.done:
			return
		default:
		}

		delay := calculateBackoff(m.backoffInit, m.backoffMax, m.backoffMul, attempt)
		m.logger.Warn("reconnecting to amqp broker",
			"attempt", attempt+1,
			"delay", delay,
		)
		time.Sleep(delay)

		if err := m.connect(); err != nil {
			m.logger.Error("reconnection failed", "attempt", attempt+1, "error", err)
			continue
		}

		m.logger.Info("reconnected to amqp broker", "attempts", attempt+1)

		if m.onReconnect != nil {
			if err := m.onReconnect(); err != nil {
				m.logger.Error("post-reconnect callback failed", "error", err)
			}
		}
		return
	}
}

// channel returns the active AMQP channel or ErrConnection if not connected.
func (m *connectionManager) channel() (*amqp.Channel, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.ch == nil || m.ch.IsClosed() {
		return nil, ErrConnection
	}
	return m.ch, nil
}

// newChannel opens a new dedicated channel on the current connection.
func (m *connectionManager) newChannel() (*amqp.Channel, error) {
	m.mu.RLock()
	conn := m.conn
	m.mu.RUnlock()
	if conn == nil || conn.IsClosed() {
		return nil, ErrConnection
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrChannel, err)
	}
	return ch, nil
}

// close shuts down the connection manager.
func (m *connectionManager) close() {
	m.stopOnce.Do(func() {
		close(m.done)

		m.mu.Lock()
		defer m.mu.Unlock()

		if m.ch != nil {
			_ = m.ch.Close()
		}
		if m.conn != nil {
			_ = m.conn.Close()
		}
	})
}

// calculateBackoff returns the backoff duration for the given attempt.
func calculateBackoff(init, max time.Duration, mul float64, attempt int) time.Duration {
	delay := float64(init) * math.Pow(mul, float64(attempt))
	if delay > float64(max) {
		return max
	}
	return time.Duration(delay)
}
