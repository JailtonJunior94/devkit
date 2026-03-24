package migrate

import (
	gmigrate "github.com/golang-migrate/migrate/v4"
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
