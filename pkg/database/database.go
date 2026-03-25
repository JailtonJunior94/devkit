// Package database provides a managed database connection pool for relational databases.
// It supports Postgres, MySQL, and SQL Server via configuration, without requiring
// code changes in the consumer when switching drivers.
//
// The consumer is responsible for registering the appropriate database driver via
// import side-effect before calling New:
//
//	import _ "devkit/pkg/database/postgres"
//	import _ "devkit/pkg/database/mysql"
//	import _ "devkit/pkg/database/sqlserver"
package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"devkit/pkg/database/internal/driverreg"
)

const (
	// DefaultMaxOpenConns is the default maximum number of open connections.
	DefaultMaxOpenConns = 25
	// DefaultMaxIdleConns is the default maximum number of idle connections.
	DefaultMaxIdleConns = 5
	// DefaultConnMaxLifetime is the default maximum time a connection may be reused.
	DefaultConnMaxLifetime = 5 * time.Minute
	// DefaultConnMaxIdleTime is the default maximum time a connection may remain idle.
	DefaultConnMaxIdleTime = 5 * time.Minute
)

// sqlOpenFn is the function used to open a database connection.
// It is a variable to allow substitution in tests.
var sqlOpenFn = sql.Open

type dbCloser interface {
	Close() error
}

// Config holds the connection parameters for the Manager.
type Config struct {
	// Driver specifies the database driver registered by a sub-package such as
	// devkit/pkg/database/postgres, devkit/pkg/database/mysql, or
	// devkit/pkg/database/sqlserver.
	Driver string

	// DSN is the connection string. It is never logged or included in error messages.
	DSN string

	// MaxOpenConns is the maximum number of open connections. Zero uses the default.
	MaxOpenConns int

	// MaxIdleConns is the maximum number of idle connections. Zero uses the default.
	MaxIdleConns int

	// ConnMaxLifetime is the maximum reuse duration for a connection. Zero uses the default.
	ConnMaxLifetime time.Duration

	// ConnMaxIdleTime is the maximum idle duration for a connection. Zero uses the default.
	ConnMaxIdleTime time.Duration
}

// Option configures optional Manager settings. Use the With... functions to
// build options and pass them to New.
type Option func(*poolOptions)

// poolOptions holds the optional connection pool settings applied by Options.
type poolOptions struct {
	maxOpenConns    int
	maxIdleConns    int
	connMaxLifetime time.Duration
	connMaxIdleTime time.Duration
}

// WithMaxOpenConns sets the maximum number of open connections to the database.
// Zero is replaced with DefaultMaxOpenConns (25).
func WithMaxOpenConns(n int) Option {
	return func(o *poolOptions) { o.maxOpenConns = n }
}

// WithMaxIdleConns sets the maximum number of idle connections retained in the pool.
// Zero is replaced with DefaultMaxIdleConns (5).
func WithMaxIdleConns(n int) Option {
	return func(o *poolOptions) { o.maxIdleConns = n }
}

// WithConnMaxLifetime sets the maximum amount of time a connection may be reused.
// Zero is replaced with DefaultConnMaxLifetime (5m).
func WithConnMaxLifetime(d time.Duration) Option {
	return func(o *poolOptions) { o.connMaxLifetime = d }
}

// WithConnMaxIdleTime sets the maximum amount of time a connection may remain idle.
// Zero is replaced with DefaultConnMaxIdleTime (5m).
func WithConnMaxIdleTime(d time.Duration) Option {
	return func(o *poolOptions) { o.connMaxIdleTime = d }
}

// Manager manages a database connection pool.
// It is safe for concurrent use; the underlying *sql.DB provides concurrency safety.
type Manager struct {
	db        *sql.DB
	closer    dbCloser
	once      sync.Once
	closeDone chan struct{}
	closeErr  error
}

// New creates a Manager, applies default pool settings, applies any provided
// Options, validates the configuration, opens a connection pool, and pings the
// database to verify connectivity. Returns an error if configuration is invalid
// or the database is unreachable.
func New(ctx context.Context, cfg Config, opts ...Option) (*Manager, error) {
	o := &poolOptions{
		maxOpenConns:    cfg.MaxOpenConns,
		maxIdleConns:    cfg.MaxIdleConns,
		connMaxLifetime: cfg.ConnMaxLifetime,
		connMaxIdleTime: cfg.ConnMaxIdleTime,
	}
	applyDefaults(o)
	for _, opt := range opts {
		opt(o)
	}

	if err := validate(cfg, *o); err != nil {
		return nil, err
	}

	db, err := sqlOpenFn(cfg.Driver, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("database: open: %w", err)
	}

	db.SetMaxOpenConns(o.maxOpenConns)
	db.SetMaxIdleConns(o.maxIdleConns)
	db.SetConnMaxLifetime(o.connMaxLifetime)
	db.SetConnMaxIdleTime(o.connMaxIdleTime)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("database: ping: %w", err)
	}

	return &Manager{
		db:        db,
		closer:    db,
		closeDone: make(chan struct{}),
	}, nil
}

// DB returns the underlying *sql.DB for direct query access.
func (m *Manager) DB() *sql.DB {
	return m.db
}

// Close closes the database connection pool idempotently and waits for either
// completion or context cancellation.
func (m *Manager) Close(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	m.once.Do(func() {
		go func() {
			m.closeErr = m.closer.Close()
			close(m.closeDone)
		}()
	})

	select {
	case <-m.closeDone:
		return m.closeErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

func applyDefaults(o *poolOptions) {
	if o.maxOpenConns == 0 {
		o.maxOpenConns = DefaultMaxOpenConns
	}
	if o.maxIdleConns == 0 {
		o.maxIdleConns = DefaultMaxIdleConns
	}
	if o.connMaxLifetime == 0 {
		o.connMaxLifetime = DefaultConnMaxLifetime
	}
	if o.connMaxIdleTime == 0 {
		o.connMaxIdleTime = DefaultConnMaxIdleTime
	}
}

func validate(cfg Config, o poolOptions) error {
	if cfg.Driver == "" {
		return ErrDriverRequired
	}
	if cfg.DSN == "" {
		return ErrDSNRequired
	}
	if !driverreg.IsRegistered(cfg.Driver) {
		return fmt.Errorf("%w: %s", ErrUnsupportedDriver, cfg.Driver)
	}
	if o.maxOpenConns < 0 || o.maxIdleConns < 0 {
		return ErrInvalidPoolConfig
	}
	if o.maxIdleConns > o.maxOpenConns {
		return ErrInvalidPoolConfig
	}
	return nil
}
