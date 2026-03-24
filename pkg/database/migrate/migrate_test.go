package migrate_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"testing"
	"testing/fstest"

	gmigrate "github.com/golang-migrate/migrate/v4"
	gmigdbstub "github.com/golang-migrate/migrate/v4/database/stub"
	gmigsource "github.com/golang-migrate/migrate/v4/source"
	gmigsrcstub "github.com/golang-migrate/migrate/v4/source/stub"
	sqlmock "github.com/DATA-DOG/go-sqlmock"

	"devkit/pkg/database/migrate"
)

// validFS is a minimal non-nil fs.FS used to pass input validation.
// Up/Down operations are tested via integration tests (build tag: integration).
var validFS fs.FS = fstest.MapFS{}

// --- New: input validation (table-driven) ---

func TestNew_inputValidation(t *testing.T) {
	db, _, _ := sqlmock.New()

	tests := []struct {
		name    string
		db      *sql.DB
		fsys    fs.FS
		cfg     migrate.Config
		wantErr error
	}{
		{
			name:    "nil db returns ErrDatabaseRequired",
			db:      nil,
			fsys:    validFS,
			cfg:     migrate.Config{DatabaseDriver: "postgres"},
			wantErr: migrate.ErrDatabaseRequired,
		},
		{
			name:    "nil fsys returns ErrSourceRequired",
			db:      db,
			fsys:    nil,
			cfg:     migrate.Config{DatabaseDriver: "postgres"},
			wantErr: migrate.ErrSourceRequired,
		},
		{
			name:    "empty driver returns ErrDatabaseRequired",
			db:      db,
			fsys:    validFS,
			cfg:     migrate.Config{DatabaseDriver: ""},
			wantErr: migrate.ErrDatabaseRequired,
		},
		{
			name:    "unknown driver returns error",
			db:      db,
			fsys:    validFS,
			cfg:     migrate.Config{DatabaseDriver: "oracle"},
			wantErr: nil, // no sentinel — just a non-nil error from createDatabaseDriver
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := migrate.New(tt.db, tt.fsys, tt.cfg)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("errors.Is mismatch: got %q, want %v", err, tt.wantErr)
			}
		})
	}
}

// --- Sentinel errors inspectable via errors.Is ---

func TestSentinelErrors_errorsIs(t *testing.T) {
	errs := []error{
		migrate.ErrSourceRequired,
		migrate.ErrDatabaseRequired,
		migrate.ErrDirtyDatabase,
	}
	for _, sentinel := range errs {
		if !errors.Is(sentinel, sentinel) {
			t.Errorf("errors.Is(%v, %v) returned false", sentinel, sentinel)
		}
	}
}

// --- wrapMigrateError ---

func TestWrapMigrateError_dirtyState(t *testing.T) {
	dirty := gmigrate.ErrDirty{Version: 7}
	err := migrate.WrapMigrateError(dirty, "up")

	if !errors.Is(err, migrate.ErrDirtyDatabase) {
		t.Errorf("expected errors.Is(err, ErrDirtyDatabase), got %v", err)
	}
	if err.Error() == "" {
		t.Error("expected non-empty error message")
	}
}

func TestWrapMigrateError_genericError(t *testing.T) {
	cause := fmt.Errorf("some db error")
	err := migrate.WrapMigrateError(cause, "down")

	if errors.Is(err, migrate.ErrDirtyDatabase) {
		t.Error("expected non-dirty error, got ErrDirtyDatabase")
	}
	if !errors.Is(err, cause) {
		t.Errorf("expected errors.Is(err, cause) to be true, got %v", err)
	}
}

// --- Up / Down / Close with stub drivers ---

// newStubMigrator creates a Migrator backed by in-memory stub drivers so that
// Up/Down/Close can be exercised in unit tests without a real database.
func newStubMigrator(t *testing.T, migrations *gmigsource.Migrations) *migrate.Migrator {
	t.Helper()

	srcDriver, err := gmigsrcstub.WithInstance(nil, &gmigsrcstub.Config{})
	if err != nil {
		t.Fatalf("source stub: %v", err)
	}
	srcDriver.(*gmigsrcstub.Stub).Migrations = migrations

	dbDriver, err := gmigdbstub.WithInstance(nil, &gmigdbstub.Config{})
	if err != nil {
		t.Fatalf("db stub: %v", err)
	}

	instance, err := gmigrate.NewWithInstance("stub", srcDriver, "stub", dbDriver)
	if err != nil {
		t.Fatalf("gmigrate.NewWithInstance: %v", err)
	}
	return migrate.NewMigratorForTest(instance)
}

func emptyMigrations() *gmigsource.Migrations {
	return gmigsource.NewMigrations()
}

func oneMigration() *gmigsource.Migrations {
	m := gmigsource.NewMigrations()
	_ = m.Append(&gmigsource.Migration{Version: 1, Direction: gmigsource.Up, Identifier: "CREATE TABLE t (id INT)"})
	_ = m.Append(&gmigsource.Migration{Version: 1, Direction: gmigsource.Down, Identifier: "DROP TABLE t"})
	return m
}

func TestUp_applyAndIdempotent(t *testing.T) {
	m := newStubMigrator(t, oneMigration())

	if err := m.Up(context.Background()); err != nil {
		t.Fatalf("first Up: %v", err)
	}
	// Second Up must be a no-op (ErrNoChange treated as nil).
	if err := m.Up(context.Background()); err != nil {
		t.Errorf("second Up (idempotent): expected nil, got %v", err)
	}
}

func TestDown_noMigrations_noError(t *testing.T) {
	m := newStubMigrator(t, emptyMigrations())
	if err := m.Down(context.Background()); err != nil {
		t.Errorf("Down with no migrations: expected nil, got %v", err)
	}
}

func TestDown_afterUp(t *testing.T) {
	m := newStubMigrator(t, oneMigration())

	if err := m.Up(context.Background()); err != nil {
		t.Fatalf("Up: %v", err)
	}
	if err := m.Down(context.Background()); err != nil {
		t.Errorf("Down after Up: %v", err)
	}
}

func TestClose_releasesResources(t *testing.T) {
	m := newStubMigrator(t, emptyMigrations())
	if err := m.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}
