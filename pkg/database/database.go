// Package database provides a managed database connection pool for relational databases.
// It supports Postgres, MySQL, and SQL Server via configuration, without requiring
// code changes in the consumer when switching drivers.
//
// The consumer is responsible for registering the appropriate database driver via
// import side-effect before calling New:
//
//	import _ "github.com/lib/pq"                     // postgres
//	import _ "github.com/go-sql-driver/mysql"         // mysql
//	import _ "github.com/microsoft/go-mssqldb"        // sqlserver
package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"
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

// supportedDrivers is the set of drivers accepted by New. Adding a new driver
// requires a single-line entry here plus its import side-effect in the consumer.
// This is intentional: the core validates eagerly rather than discovering drivers
// from the global sql registry, which could silently accept unintended drivers.
var supportedDrivers = map[string]bool{
	"postgres":  true,
	"mysql":     true,
	"sqlserver": true,
}

// sqlOpenFn is the function used to open a database connection.
// It is a variable to allow substitution in tests.
var sqlOpenFn = sql.Open

// Config holds the required connection parameters for the Manager.
// Optional pool settings are configured via Option functions passed to New.
type Config struct {
	// Driver specifies the database driver: "postgres", "mysql", or "sqlserver".
	// The consumer must register the driver via import side-effect.
	Driver string

	// DSN is the connection string. It is never logged or included in error messages.
	DSN string
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
	db   *sql.DB
	once sync.Once
}

// New creates a Manager, applies default pool settings, applies any provided
// Options, validates the configuration, opens a connection pool, and pings the
// database to verify connectivity. Returns an error if configuration is invalid
// or the database is unreachable.
func New(ctx context.Context, cfg Config, opts ...Option) (*Manager, error) {
	o := &poolOptions{}
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

	return &Manager{db: db}, nil
}

// DB returns the underlying *sql.DB for direct query access.
func (m *Manager) DB() *sql.DB {
	return m.db
}

// Close closes the database connection pool idempotently.
// The first call closes the pool; subsequent calls are no-ops returning nil.
//
// Note: the context parameter is accepted for API consistency but is not
// propagated to *sql.DB.Close — the standard library does not expose a
// context-aware close. Consumers that require a hard timeout on shutdown
// should wrap this call with their own context cancellation logic.
func (m *Manager) Close(_ context.Context) error {
	var err error
	m.once.Do(func() {
		err = m.db.Close()
	})
	return err
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
	if !supportedDrivers[cfg.Driver] {
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
