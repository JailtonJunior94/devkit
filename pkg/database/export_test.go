package database

import "database/sql"

// NewFromDB creates a Manager from an existing *sql.DB without validation or ping.
// This function is only available during test compilation.
func NewFromDB(db *sql.DB) *Manager {
	return &Manager{
		db:        db,
		closer:    db,
		closeDone: make(chan struct{}),
	}
}

// SetSQLOpenFunc replaces the sqlOpenFn for the duration of a test.
// It returns a restore function that reverts to the original.
// This function is only available during test compilation.
func SetSQLOpenFunc(fn func(driverName, dataSourceName string) (*sql.DB, error)) func() {
	orig := sqlOpenFn
	sqlOpenFn = fn
	return func() { sqlOpenFn = orig }
}

// NewWithCloser creates a Manager with a custom closer for tests that need to
// control shutdown timing.
func NewWithCloser(db *sql.DB, closer interface{ Close() error }) *Manager {
	return &Manager{
		db:        db,
		closer:    closer,
		closeDone: make(chan struct{}),
	}
}
