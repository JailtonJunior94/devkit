package database_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"

	"devkit/pkg/database"
	_ "devkit/pkg/database/mysql"
	_ "devkit/pkg/database/postgres"
	_ "devkit/pkg/database/sqlserver"
)

// --- Config validation (table-driven) ---

func TestNew_configValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     database.Config
		opts    []database.Option
		wantErr error
	}{
		{
			name:    "empty driver",
			cfg:     database.Config{Driver: "", DSN: "postgres://localhost/testdb"},
			wantErr: database.ErrDriverRequired,
		},
		{
			name:    "empty DSN",
			cfg:     database.Config{Driver: "postgres", DSN: ""},
			wantErr: database.ErrDSNRequired,
		},
		{
			name:    "unsupported driver oracle",
			cfg:     database.Config{Driver: "oracle", DSN: "oracle://localhost"},
			wantErr: database.ErrUnsupportedDriver,
		},
		{
			name: "max idle conns greater than max open conns",
			cfg: database.Config{
				Driver:       "postgres",
				DSN:          "postgres://localhost/testdb",
				MaxOpenConns: 5,
				MaxIdleConns: 10,
			},
			wantErr: database.ErrInvalidPoolConfig,
		},
		{
			name: "negative max open conns",
			cfg: database.Config{
				Driver:       "postgres",
				DSN:          "postgres://localhost/testdb",
				MaxOpenConns: -1,
			},
			wantErr: database.ErrInvalidPoolConfig,
		},
		{
			name: "negative max idle conns",
			cfg: database.Config{
				Driver:       "postgres",
				DSN:          "postgres://localhost/testdb",
				MaxIdleConns: -1,
			},
			wantErr: database.ErrInvalidPoolConfig,
		},
		{
			name: "max idle conns exceeds default max open conns",
			cfg: database.Config{
				Driver:       "postgres",
				DSN:          "postgres://localhost/testdb",
				MaxIdleConns: database.DefaultMaxOpenConns + 1,
			},
			wantErr: database.ErrInvalidPoolConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := database.New(context.Background(), tt.cfg, tt.opts...)
			if err == nil {
				t.Fatalf("expected error %v, got nil", tt.wantErr)
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("errors.Is mismatch: got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// --- Config defaults ---

func TestNew_defaultsApplied(t *testing.T) {
	cfg := database.Config{
		Driver:       "postgres",
		DSN:          "postgres://localhost/testdb",
		MaxIdleConns: database.DefaultMaxOpenConns + 1,
	}
	_, err := database.New(context.Background(), cfg)
	if !errors.Is(err, database.ErrInvalidPoolConfig) {
		t.Fatalf("expected ErrInvalidPoolConfig (proving MaxOpenConns defaulted to %d), got %v",
			database.DefaultMaxOpenConns, err)
	}

	// Verify exported default constants have the documented values.
	if database.DefaultMaxOpenConns != 25 {
		t.Errorf("DefaultMaxOpenConns = %d, want 25", database.DefaultMaxOpenConns)
	}
	if database.DefaultMaxIdleConns != 5 {
		t.Errorf("DefaultMaxIdleConns = %d, want 5", database.DefaultMaxIdleConns)
	}
	if database.DefaultConnMaxLifetime != 5*time.Minute {
		t.Errorf("DefaultConnMaxLifetime = %v, want 5m", database.DefaultConnMaxLifetime)
	}
	if database.DefaultConnMaxIdleTime != 5*time.Minute {
		t.Errorf("DefaultConnMaxIdleTime = %v, want 5m", database.DefaultConnMaxIdleTime)
	}
}

func TestNew_configPoolFieldsApplied(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}

	restore := database.SetSQLOpenFunc(func(_, _ string) (*sql.DB, error) {
		return db, nil
	})
	defer restore()

	mgr, err := database.New(context.Background(), database.Config{
		Driver:          "postgres",
		DSN:             "postgres://localhost/testdb",
		MaxOpenConns:    7,
		MaxIdleConns:    3,
		ConnMaxLifetime: 2 * time.Minute,
		ConnMaxIdleTime: time.Minute,
	})
	if err != nil {
		t.Fatalf("database.New: %v", err)
	}

	stats := mgr.DB().Stats()
	if stats.MaxOpenConnections != 7 {
		t.Fatalf("MaxOpenConnections = %d, want 7", stats.MaxOpenConnections)
	}
}

func TestNew_optionsOverrideConfigFields(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}

	restore := database.SetSQLOpenFunc(func(_, _ string) (*sql.DB, error) {
		return db, nil
	})
	defer restore()

	mgr, err := database.New(context.Background(), database.Config{
		Driver:       "postgres",
		DSN:          "postgres://localhost/testdb",
		MaxOpenConns: 7,
		MaxIdleConns: 3,
	}, database.WithMaxOpenConns(11), database.WithMaxIdleConns(4))
	if err != nil {
		t.Fatalf("database.New: %v", err)
	}

	stats := mgr.DB().Stats()
	if stats.MaxOpenConnections != 11 {
		t.Fatalf("MaxOpenConnections = %d, want 11", stats.MaxOpenConnections)
	}
}

// --- New with sqlmock ---

func TestNew_success(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}

	restore := database.SetSQLOpenFunc(func(_, _ string) (*sql.DB, error) {
		return db, nil
	})
	defer restore()

	mgr, err := database.New(context.Background(), database.Config{
		Driver: "postgres",
		DSN:    "postgres://localhost/testdb",
	})
	if err != nil {
		t.Fatalf("database.New: %v", err)
	}
	if mgr == nil {
		t.Fatal("expected non-nil Manager")
	}
	if mgr.DB() == nil {
		t.Fatal("expected non-nil *sql.DB from Manager.DB()")
	}
}

func TestNew_pingFailure(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}

	restore := database.SetSQLOpenFunc(func(_, _ string) (*sql.DB, error) {
		return db, nil
	})
	defer restore()

	// A canceled context causes PingContext to fail immediately.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = database.New(ctx, database.Config{
		Driver: "postgres",
		DSN:    "postgres://localhost/testdb",
	})
	if err == nil {
		t.Fatal("expected ping error, got nil")
	}
}

// --- Close idempotence ---

func TestClose_idempotent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	// Expect exactly one Close call.
	mock.ExpectClose()

	mgr := database.NewFromDB(db)
	ctx := context.Background()

	if err := mgr.Close(ctx); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	// Second Close must be a no-op (idempotent).
	if err := mgr.Close(ctx); err != nil {
		t.Fatalf("second Close: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

type blockingCloser struct {
	started chan struct{}
	release chan struct{}
}

func (c *blockingCloser) Close() error {
	close(c.started)
	<-c.release
	return nil
}

func TestClose_respectsContextCancellation(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}

	closer := &blockingCloser{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	mgr := database.NewWithCloser(db, closer)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- mgr.Close(ctx)
	}()

	<-closer.started

	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Close error = %v, want %v", err, context.DeadlineExceeded)
		}
	case <-time.After(time.Second):
		t.Fatal("Close did not return after context cancellation")
	}

	close(closer.release)

	if err := mgr.Close(context.Background()); err != nil {
		t.Fatalf("Close after release: %v", err)
	}
}

// --- DB accessor ---

func TestDB_returnsUnderlyingDB(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	mgr := database.NewFromDB(db)
	if mgr.DB() != db {
		t.Error("DB() did not return the underlying *sql.DB")
	}
}

// --- Sentinel errors are inspectable via errors.Is ---

func TestSentinelErrors_errorsIs(t *testing.T) {
	errs := []error{
		database.ErrDriverRequired,
		database.ErrDSNRequired,
		database.ErrUnsupportedDriver,
		database.ErrInvalidPoolConfig,
	}
	for _, sentinel := range errs {
		if !errors.Is(sentinel, sentinel) {
			t.Errorf("errors.Is(%v, %v) returned false", sentinel, sentinel)
		}
	}
}
