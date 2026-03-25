//go:build integration

package migrate_test

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"
	"os"
	"testing"

	testcontainers "github.com/testcontainers/testcontainers-go"
	tcmssql "github.com/testcontainers/testcontainers-go/modules/mssql"
	tcmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	_ "devkit/pkg/database/mysql"
	_ "devkit/pkg/database/postgres"
	_ "devkit/pkg/database/sqlserver"

	"devkit/pkg/database/migrate"
)

//go:embed testdata/migrations/postgres
var postgresFS embed.FS

//go:embed testdata/migrations/mysql
var mysqlFS embed.FS

//go:embed testdata/migrations/sqlserver
var sqlserverFS embed.FS

type containerSetup struct {
	driver string
	dsn    string
}

func startPostgres(t *testing.T) containerSetup {
	t.Helper()
	ctx := context.Background()
	c, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() { testcontainers.TerminateContainer(c) })
	dsn, err := c.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	return containerSetup{driver: "postgres", dsn: dsn}
}

func startMySQL(t *testing.T) containerSetup {
	t.Helper()
	ctx := context.Background()
	c, err := tcmysql.Run(ctx, "mysql:8.0",
		tcmysql.WithDatabase("testdb"),
		tcmysql.WithUsername("test"),
		tcmysql.WithPassword("test"),
	)
	if err != nil {
		t.Fatalf("start mysql container: %v", err)
	}
	t.Cleanup(func() { testcontainers.TerminateContainer(c) })
	dsn, err := c.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	return containerSetup{driver: "mysql", dsn: dsn}
}

func startSQLServer(t *testing.T) containerSetup {
	t.Helper()
	ctx := context.Background()
	c, err := tcmssql.Run(ctx, "mcr.microsoft.com/mssql/server:2022-latest",
		tcmssql.WithPassword("Test@1234"),
		tcmssql.WithAcceptEULA(),
	)
	if err != nil {
		t.Fatalf("start mssql container: %v", err)
	}
	t.Cleanup(func() { testcontainers.TerminateContainer(c) })
	dsn, err := c.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	return containerSetup{driver: "sqlserver", dsn: dsn}
}

func openTestDB(t *testing.T, setup containerSetup) *sql.DB {
	t.Helper()
	db, err := sql.Open(setup.driver, setup.dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}
	return db
}

func migrationsFS(t *testing.T, driver string) fs.FS {
	t.Helper()
	switch driver {
	case "postgres":
		sub, err := fs.Sub(postgresFS, "testdata/migrations/postgres")
		if err != nil {
			t.Fatalf("sub fs postgres: %v", err)
		}
		return sub
	case "mysql":
		sub, err := fs.Sub(mysqlFS, "testdata/migrations/mysql")
		if err != nil {
			t.Fatalf("sub fs mysql: %v", err)
		}
		return sub
	case "sqlserver":
		sub, err := fs.Sub(sqlserverFS, "testdata/migrations/sqlserver")
		if err != nil {
			t.Fatalf("sub fs sqlserver: %v", err)
		}
		return sub
	default:
		t.Fatalf("unknown driver: %s", driver)
		return nil
	}
}

func runMigrateTests(t *testing.T, setup containerSetup) {
	t.Helper()
	db := openTestDB(t, setup)
	ctx := context.Background()
	fsys := migrationsFS(t, setup.driver)

	m, err := migrate.New(db, fsys, migrate.Config{DatabaseDriver: setup.driver})
	if err != nil {
		t.Fatalf("migrate.New: %v", err)
	}
	t.Cleanup(func() { m.Close() })

	t.Run("up_creates_schema", func(t *testing.T) {
		if err := m.Up(ctx); err != nil {
			t.Fatalf("Up: %v", err)
		}
		var count int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count); err != nil {
			t.Errorf("query after Up: %v", err)
		}
	})

	t.Run("up_idempotent_no_change", func(t *testing.T) {
		// Up after schema is already up to date must not error.
		if err := m.Up(ctx); err != nil {
			t.Errorf("second Up (ErrNoChange treated as success): %v", err)
		}
	})

	t.Run("down_removes_schema", func(t *testing.T) {
		if err := m.Down(ctx); err != nil {
			t.Fatalf("Down: %v", err)
		}
		var count int
		err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
		if err == nil {
			t.Error("expected error querying dropped table, got nil")
		}
	})
}

func TestMigratePostgres(t *testing.T) { runMigrateTests(t, startPostgres(t)) }
func TestMigrateMySQL(t *testing.T)    { runMigrateTests(t, startMySQL(t)) }
func TestMigrateSQLServer(t *testing.T) {
	if os.Getenv("RUN_SQLSERVER_INTEGRATION") != "1" {
		t.Skip("SQL Server requires ~1.5GB Docker image; set RUN_SQLSERVER_INTEGRATION=1 to run manually")
	}
	runMigrateTests(t, startSQLServer(t))
}
