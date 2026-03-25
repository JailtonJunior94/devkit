//go:build integration

package database_test

import (
	"context"
	"testing"
	"testing/fstest"

	testcontainers "github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"devkit/pkg/database"
	"devkit/pkg/database/migrate"
	_ "devkit/pkg/database/postgres"
	"devkit/pkg/database/uow"
)

func TestStackCompletePostgres(t *testing.T) {
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() { testcontainers.TerminateContainer(container) })

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	mgr, err := database.New(ctx, database.Config{Driver: "postgres", DSN: dsn})
	if err != nil {
		t.Fatalf("database.New: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close(ctx) })

	migrations := fstest.MapFS{
		"1_init.up.sql": {
			Data: []byte("CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT NOT NULL);"),
		},
		"1_init.down.sql": {
			Data: []byte("DROP TABLE users;"),
		},
	}

	migrator, err := migrate.New(mgr.DB(), migrations, migrate.Config{DatabaseDriver: "postgres"})
	if err != nil {
		t.Fatalf("migrate.New: %v", err)
	}
	t.Cleanup(func() { _ = migrator.Close() })

	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("migrator.Up: %v", err)
	}

	unit, err := uow.New(mgr.DB())
	if err != nil {
		t.Fatalf("uow.New: %v", err)
	}

	if err := unit.Do(ctx, func(ctx context.Context) error {
		tx := uow.TxFromContext(ctx)
		_, err := tx.ExecContext(ctx, "INSERT INTO users (name) VALUES ($1)", "Alice")
		return err
	}); err != nil {
		t.Fatalf("uow.Do: %v", err)
	}

	var count int
	if err := mgr.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE name = $1", "Alice").Scan(&count); err != nil {
		t.Fatalf("query inserted row: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}
