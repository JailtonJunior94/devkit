package database_test

import (
	"context"
	"fmt"
	"testing/fstest"

	"devkit/pkg/database"
	"devkit/pkg/database/migrate"
	_ "devkit/pkg/database/postgres"
	"devkit/pkg/database/uow"
)

// Example_stackComplete demonstrates Scenario D: composing database, migrate,
// and uow in the same startup flow.
func Example_stackComplete() {
	ctx := context.Background()

	mgr, err := database.New(ctx, database.Config{
		Driver: "postgres",
		DSN:    "postgres://user:pass@localhost/mydb?sslmode=disable",
	})
	if err != nil {
		fmt.Printf("database.New: %v\n", err)
		return
	}
	defer func() { _ = mgr.Close(ctx) }()

	migrations := fstest.MapFS{
		"1_init.up.sql":   {Data: []byte("CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT NOT NULL);")},
		"1_init.down.sql": {Data: []byte("DROP TABLE users;")},
	}

	migrator, err := migrate.New(mgr.DB(), migrations, migrate.Config{DatabaseDriver: "postgres"})
	if err != nil {
		fmt.Printf("migrate.New: %v\n", err)
		return
	}
	defer func() { _ = migrator.Close() }()

	if err := migrator.Up(ctx); err != nil {
		fmt.Printf("migrator.Up: %v\n", err)
		return
	}

	unit, err := uow.New(mgr.DB())
	if err != nil {
		fmt.Printf("uow.New: %v\n", err)
		return
	}

	err = unit.Do(ctx, func(ctx context.Context) error {
		tx := uow.TxFromContext(ctx)
		_, err := tx.ExecContext(ctx, "INSERT INTO users (name) VALUES ($1)", "Alice")
		return err
	})
	if err != nil {
		fmt.Printf("uow.Do: %v\n", err)
	}
}
