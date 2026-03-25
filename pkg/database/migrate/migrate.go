// Package migrate provides database migration support via golang-migrate.
// It is independent of pkg/database and pkg/database/uow — it only depends on
// database/sql from the stdlib. The consumer composes it with a *sql.DB obtained
// from pkg/database or created independently.
//
// Migrations are loaded from an fs.FS source, which supports embed.FS for
// embedding SQL files in the binary:
//
//	//go:embed migrations
//	var migrationsFS embed.FS
//
//	m, err := migrate.New(db, migrationsFS, migrate.Config{DatabaseDriver: "postgres"})
//	if err != nil { ... }
//	defer m.Close()
//	if err := m.Up(ctx); err != nil { ... }
package migrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"

	gmigrate "github.com/golang-migrate/migrate/v4"
	gmigdatabase "github.com/golang-migrate/migrate/v4/database"
	gmigmysql "github.com/golang-migrate/migrate/v4/database/mysql"
	gmigpostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	gmigsqlserver "github.com/golang-migrate/migrate/v4/database/sqlserver"
	gmigsource "github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

var (
	newSourceDriver        func(fs.FS, string) (gmigsource.Driver, error)                                          = iofs.New
	newMigrateWithInstance func(string, gmigsource.Driver, string, gmigdatabase.Driver) (*gmigrate.Migrate, error) = gmigrate.NewWithInstance
	createPostgresDriver                                                                                           = func(db *sql.DB, migrationsTable string) (gmigdatabase.Driver, error) {
		return gmigpostgres.WithInstance(db, &gmigpostgres.Config{MigrationsTable: migrationsTable})
	}
	createMySQLDriver = func(db *sql.DB, migrationsTable string) (gmigdatabase.Driver, error) {
		return gmigmysql.WithInstance(db, &gmigmysql.Config{MigrationsTable: migrationsTable})
	}
	createSQLServerDriver = func(db *sql.DB, migrationsTable string) (gmigdatabase.Driver, error) {
		return gmigsqlserver.WithInstance(db, &gmigsqlserver.Config{MigrationsTable: migrationsTable})
	}
)

// Option configures optional Migrator settings. Use the With... functions to
// build options and pass them to New.
type Option func(*migrateOptions)

// migrateOptions holds the optional settings applied by Options.
type migrateOptions struct {
	migrationsTable string
}

// WithMigrationsTable sets the name of the table used to track applied
// migrations. Default: "" (each driver uses its own default, typically
// "schema_migrations").
func WithMigrationsTable(table string) Option {
	return func(o *migrateOptions) { o.migrationsTable = table }
}

func applyDefaults(_ *migrateOptions) {
	// Default migrationsTable is "": each driver falls back to its own default
	// ("schema_migrations" for postgres/mysql, "schema_migrations" for sqlserver).
}

// Config holds the required configuration to build a Migrator.
type Config struct {
	// DatabaseDriver specifies the target database driver: "postgres", "mysql",
	// or "sqlserver". It must match the driver used to open the *sql.DB.
	DatabaseDriver string
}

// Migrator wraps golang-migrate for Up/Down migration operations.
// It is created from a *sql.DB and an fs.FS containing SQL migration files.
type Migrator struct {
	instance *gmigrate.Migrate
}

// New creates a Migrator from a *sql.DB and an fs.FS migration source.
// fsys must contain migration files in the root directory (".").
//
// Returns ErrDatabaseRequired if db is nil or cfg.DatabaseDriver is empty.
// Returns ErrSourceRequired if fsys is nil.
func New(db *sql.DB, fsys fs.FS, cfg Config, opts ...Option) (*Migrator, error) {
	if db == nil {
		return nil, ErrDatabaseRequired
	}
	if fsys == nil {
		return nil, ErrSourceRequired
	}
	if cfg.DatabaseDriver == "" {
		return nil, ErrDatabaseRequired
	}

	o := &migrateOptions{}
	applyDefaults(o)
	for _, opt := range opts {
		opt(o)
	}

	sourceDriver, err := newSourceDriver(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("migrate: create source driver: %w", err)
	}

	dbDriver, err := createDatabaseDriver(db, cfg.DatabaseDriver, o.migrationsTable)
	if err != nil {
		return nil, fmt.Errorf("migrate: create database driver: %w", err)
	}

	instance, err := newMigrateWithInstance("iofs", sourceDriver, cfg.DatabaseDriver, dbDriver)
	if err != nil {
		return nil, fmt.Errorf("migrate: create instance: %w", err)
	}

	return &Migrator{instance: instance}, nil
}

// Up applies all pending migrations.
// If there are no pending migrations, Up returns nil (ErrNoChange is treated
// as success). Context is accepted for API consistency but is not propagated
// to the underlying golang-migrate instance.
func (m *Migrator) Up(_ context.Context) error {
	if err := m.instance.Up(); err != nil && !errors.Is(err, gmigrate.ErrNoChange) {
		return wrapMigrateError(err, "up")
	}
	return nil
}

// Down rolls back all applied migrations.
// If there are no applied migrations, Down returns nil (ErrNoChange is treated
// as success). Context is accepted for API consistency but is not propagated
// to the underlying golang-migrate instance.
func (m *Migrator) Down(_ context.Context) error {
	if err := m.instance.Down(); err != nil && !errors.Is(err, gmigrate.ErrNoChange) {
		return wrapMigrateError(err, "down")
	}
	return nil
}

// Close releases resources held by the Migrator. It should be called when
// the Migrator is no longer needed, typically via defer.
func (m *Migrator) Close() error {
	srcErr, dbErr := m.instance.Close()
	return errors.Join(srcErr, dbErr)
}

func wrapMigrateError(err error, direction string) error {
	var dirtyErr gmigrate.ErrDirty
	if errors.As(err, &dirtyErr) {
		return fmt.Errorf("%w: version %d", ErrDirtyDatabase, dirtyErr.Version)
	}
	return fmt.Errorf("migrate: %s: %w", direction, err)
}

func createDatabaseDriver(db *sql.DB, driver, migrationsTable string) (gmigdatabase.Driver, error) {
	switch driver {
	case "postgres":
		return createPostgresDriver(db, migrationsTable)
	case "mysql":
		return createMySQLDriver(db, migrationsTable)
	case "sqlserver":
		return createSQLServerDriver(db, migrationsTable)
	default:
		return nil, fmt.Errorf("migrate: unsupported driver: %s", driver)
	}
}
