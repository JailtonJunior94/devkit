//go:build integration

package database_test

import (
	"context"
	"os"
	"testing"

	testcontainers "github.com/testcontainers/testcontainers-go"
	tcmssql "github.com/testcontainers/testcontainers-go/modules/mssql"
	tcmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"devkit/pkg/database"
	_ "devkit/pkg/database/mysql"
	_ "devkit/pkg/database/postgres"
	_ "devkit/pkg/database/sqlserver"
)

// --- Postgres ---

func TestManagerPostgres_newPingClose(t *testing.T) {
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
	defer mgr.Close(ctx)

	if err := mgr.DB().PingContext(ctx); err != nil {
		t.Errorf("PingContext: %v", err)
	}
}

func TestManagerPostgres_closeIdempotent(t *testing.T) {
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

	if err := mgr.Close(ctx); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := mgr.Close(ctx); err != nil {
		t.Fatalf("second Close (idempotent): %v", err)
	}
}

func TestManagerPostgres_poolConfig(t *testing.T) {
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

	mgr, err := database.New(ctx, database.Config{Driver: "postgres", DSN: dsn},
		database.WithMaxOpenConns(5),
		database.WithMaxIdleConns(2),
	)
	if err != nil {
		t.Fatalf("database.New with custom pool: %v", err)
	}
	defer mgr.Close(ctx)

	stats := mgr.DB().Stats()
	if stats.MaxOpenConnections != 5 {
		t.Errorf("MaxOpenConnections = %d, want 5", stats.MaxOpenConnections)
	}
}

// --- MySQL ---

func TestManagerMySQL_newPingClose(t *testing.T) {
	ctx := context.Background()

	container, err := tcmysql.Run(ctx, "mysql:8.0",
		tcmysql.WithDatabase("testdb"),
		tcmysql.WithUsername("test"),
		tcmysql.WithPassword("test"),
	)
	if err != nil {
		t.Fatalf("start mysql container: %v", err)
	}
	t.Cleanup(func() { testcontainers.TerminateContainer(container) })

	dsn, err := container.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	mgr, err := database.New(ctx, database.Config{Driver: "mysql", DSN: dsn})
	if err != nil {
		t.Fatalf("database.New: %v", err)
	}
	defer mgr.Close(ctx)

	if err := mgr.DB().PingContext(ctx); err != nil {
		t.Errorf("PingContext: %v", err)
	}
}

// --- SQL Server ---

func TestManagerSQLServer_newPingClose(t *testing.T) {
	if os.Getenv("RUN_SQLSERVER_INTEGRATION") != "1" {
		t.Skip("SQL Server requires ~1.5GB Docker image; set RUN_SQLSERVER_INTEGRATION=1 to run manually")
	}

	ctx := context.Background()

	container, err := tcmssql.Run(ctx, "mcr.microsoft.com/mssql/server:2022-latest",
		tcmssql.WithPassword("Test@1234"),
		tcmssql.WithAcceptEULA(),
	)
	if err != nil {
		t.Fatalf("start mssql container: %v", err)
	}
	t.Cleanup(func() { testcontainers.TerminateContainer(container) })

	dsn, err := container.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	mgr, err := database.New(ctx, database.Config{Driver: "sqlserver", DSN: dsn})
	if err != nil {
		t.Fatalf("database.New: %v", err)
	}
	defer mgr.Close(ctx)

	if err := mgr.DB().PingContext(ctx); err != nil {
		t.Errorf("PingContext: %v", err)
	}
}
