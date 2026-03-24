package migrate_test

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"devkit/pkg/database/migrate"
)

//go:embed testdata/migrations/postgres
var exampleMigrationsFS embed.FS

// ExampleNew demonstrates Scenario C: running migrations on startup.
// The consumer provides a *sql.DB (from database.Manager or created directly)
// and an embed.FS containing SQL migration files.
func ExampleNew() {
	var db *sql.DB // obtained from database.Manager.DB() or created directly

	sub, err := fs.Sub(exampleMigrationsFS, "testdata/migrations/postgres")
	if err != nil {
		fmt.Printf("sub fs: %v\n", err)
		return
	}

	m, err := migrate.New(db, sub, migrate.Config{DatabaseDriver: "postgres"})
	if err != nil {
		fmt.Printf("migrate.New: %v\n", err)
		return
	}
	defer func() { _ = m.Close() }()

	ctx := context.Background()
	if err := m.Up(ctx); err != nil {
		fmt.Printf("migration up failed: %v\n", err)
		return
	}
	fmt.Println("migrations applied")
}

// ExampleMigrator_Down demonstrates rolling back all migrations.
func ExampleMigrator_Down() {
	var db *sql.DB

	sub, err := fs.Sub(exampleMigrationsFS, "testdata/migrations/postgres")
	if err != nil {
		fmt.Printf("sub fs: %v\n", err)
		return
	}
	m, err := migrate.New(db, sub, migrate.Config{DatabaseDriver: "postgres"})
	if err != nil {
		fmt.Printf("migrate.New: %v\n", err)
		return
	}
	defer func() { _ = m.Close() }()

	ctx := context.Background()
	if err := m.Down(ctx); err != nil {
		fmt.Printf("migration down failed: %v\n", err)
		return
	}
	fmt.Println("migrations rolled back")
}
