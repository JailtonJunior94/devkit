package migrate

import "errors"

var (
	// ErrSourceRequired is returned when a nil fs.FS is passed to New.
	ErrSourceRequired = errors.New("migrate: source is required")

	// ErrDatabaseRequired is returned when a nil *sql.DB or an empty
	// DatabaseDriver is passed to New.
	ErrDatabaseRequired = errors.New("migrate: database config is required")

	// ErrDirtyDatabase is returned when the migration table contains a dirty
	// version marker. Use Force() on the underlying migrate instance to resolve.
	ErrDirtyDatabase = errors.New("migrate: database is in dirty state")
)
