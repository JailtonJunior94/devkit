package migrate

import (
	"database/sql"
	"io/fs"

	gmigrate "github.com/golang-migrate/migrate/v4"
	gmigdatabase "github.com/golang-migrate/migrate/v4/database"
	gmigsource "github.com/golang-migrate/migrate/v4/source"
)

// NewMigratorForTest creates a Migrator wrapping an existing *gmigrate.Migrate
// instance. This is only available during test compilation and is used to test
// Up, Down, Close, and error-wrapping logic without a real database.
func NewMigratorForTest(instance *gmigrate.Migrate) *Migrator {
	return &Migrator{instance: instance}
}

// WrapMigrateError exposes the internal wrapMigrateError for direct unit testing.
func WrapMigrateError(err error, direction string) error {
	return wrapMigrateError(err, direction)
}

// CreateDatabaseDriver exposes the internal createDatabaseDriver for unit tests.
func CreateDatabaseDriver(db *sql.DB, driver, migrationsTable string) (gmigdatabase.Driver, error) {
	return createDatabaseDriver(db, driver, migrationsTable)
}

// SetDatabaseDriverBuildersForTest replaces the driver builders for the duration
// of a test and returns a restore function.
func SetDatabaseDriverBuildersForTest(
	postgres func(*sql.DB, string) (gmigdatabase.Driver, error),
	mysql func(*sql.DB, string) (gmigdatabase.Driver, error),
	sqlserver func(*sql.DB, string) (gmigdatabase.Driver, error),
) func() {
	origPostgres := createPostgresDriver
	origMySQL := createMySQLDriver
	origSQLServer := createSQLServerDriver

	createPostgresDriver = postgres
	createMySQLDriver = mysql
	createSQLServerDriver = sqlserver

	return func() {
		createPostgresDriver = origPostgres
		createMySQLDriver = origMySQL
		createSQLServerDriver = origSQLServer
	}
}

// SetConstructorsForTest replaces source and migrate constructors for the duration
// of a test and returns a restore function.
func SetConstructorsForTest(
	source func(fs.FS, string) (gmigsource.Driver, error),
	instance func(string, gmigsource.Driver, string, gmigdatabase.Driver) (*gmigrate.Migrate, error),
) func() {
	origSource := newSourceDriver
	origInstance := newMigrateWithInstance

	newSourceDriver = func(fsys fs.FS, path string) (gmigsource.Driver, error) {
		return source(fsys, path)
	}
	newMigrateWithInstance = func(sourceName string, sourceInstance gmigsource.Driver, databaseName string, databaseInstance gmigdatabase.Driver) (*gmigrate.Migrate, error) {
		return instance(sourceName, sourceInstance, databaseName, databaseInstance)
	}

	return func() {
		newSourceDriver = origSource
		newMigrateWithInstance = origInstance
	}
}
