//go:build integration

package uow_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	testcontainers "github.com/testcontainers/testcontainers-go"
	tcmssql "github.com/testcontainers/testcontainers-go/modules/mssql"
	tcmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	_ "devkit/pkg/database/mysql"
	_ "devkit/pkg/database/postgres"
	_ "devkit/pkg/database/sqlserver"

	"devkit/pkg/database/uow"
)

// driverSetup holds the information needed to open a database connection
// in an integration test.
type driverSetup struct {
	driver string
	dsn    string
}

func setupPostgres(t *testing.T) driverSetup {
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
	return driverSetup{driver: "postgres", dsn: dsn}
}

func setupMySQL(t *testing.T) driverSetup {
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
	return driverSetup{driver: "mysql", dsn: dsn}
}

func setupSQLServer(t *testing.T) driverSetup {
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
	return driverSetup{driver: "sqlserver", dsn: dsn}
}

func openDB(t *testing.T, setup driverSetup) *sql.DB {
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

func createItemsTable(t *testing.T, db *sql.DB, driver string) {
	t.Helper()
	var createSQL string
	switch driver {
	case "postgres":
		createSQL = `CREATE TABLE IF NOT EXISTS items (id SERIAL PRIMARY KEY, name TEXT NOT NULL)`
	case "mysql":
		createSQL = `CREATE TABLE IF NOT EXISTS items (id INT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(255) NOT NULL)`
	case "sqlserver":
		createSQL = `IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='items' AND xtype='U') CREATE TABLE items (id INT IDENTITY(1,1) PRIMARY KEY, name NVARCHAR(255) NOT NULL)`
	}
	if _, err := db.Exec(createSQL); err != nil {
		t.Fatalf("create items table: %v", err)
	}
}

// runUOWTests runs the shared UOW integration scenarios against a given setup.
func runUOWTests(t *testing.T, setup driverSetup) {
	t.Helper()
	db := openDB(t, setup)
	createItemsTable(t, db, setup.driver)

	t.Run("do_commits_on_success", func(t *testing.T) {
		u, _ := uow.New(db)
		u.Register("inserter", func(tx *sql.Tx) any {
			return tx
		})

		ctx := context.Background()
		err := u.Do(ctx, func(ctx context.Context) error {
			tx := uow.TxFromContext(ctx)
			if tx == nil {
				t.Error("expected active transaction in context")
				return nil
			}
			_, err := tx.ExecContext(ctx, insertSQL(setup.driver), "committed item")
			return err
		})
		if err != nil {
			t.Fatalf("Do: %v", err)
		}

		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM items WHERE name = 'committed item'`).Scan(&count); err != nil {
			t.Fatalf("count query: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 committed row, got %d", count)
		}
	})

	t.Run("do_rollsback_on_error", func(t *testing.T) {
		u, _ := uow.New(db)
		ctx := context.Background()
		_ = u.Do(ctx, func(ctx context.Context) error {
			tx := uow.TxFromContext(ctx)
			if _, err := tx.ExecContext(ctx, insertSQL(setup.driver), "rolled back item"); err != nil {
				return err
			}
			return context.Canceled // trigger rollback
		})

		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM items WHERE name = 'rolled back item'`).Scan(&count); err != nil {
			t.Fatalf("count query: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 rows after rollback, got %d", count)
		}
	})

	t.Run("do_rollsback_on_panic", func(t *testing.T) {
		u, _ := uow.New(db)
		ctx := context.Background()

		func() {
			defer func() { recover() }()
			_ = u.Do(ctx, func(ctx context.Context) error {
				tx := uow.TxFromContext(ctx)
				if _, err := tx.ExecContext(ctx, insertSQL(setup.driver), "panicked item"); err != nil {
					return err
				}
				panic("test panic")
			})
		}()

		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM items WHERE name = 'panicked item'`).Scan(&count); err != nil {
			t.Fatalf("count query: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 rows after panic rollback, got %d", count)
		}
	})
}

func insertSQL(driver string) string {
	switch driver {
	case "postgres":
		return `INSERT INTO items (name) VALUES ($1)`
	case "sqlserver":
		return `INSERT INTO items (name) VALUES (@p1)`
	default:
		return `INSERT INTO items (name) VALUES (?)`
	}
}

func TestUOWPostgres(t *testing.T) { runUOWTests(t, setupPostgres(t)) }
func TestUOWMySQL(t *testing.T)    { runUOWTests(t, setupMySQL(t)) }
func TestUOWSQLServer(t *testing.T) {
	if os.Getenv("RUN_SQLSERVER_INTEGRATION") != "1" {
		t.Skip("SQL Server requires ~1.5GB Docker image; set RUN_SQLSERVER_INTEGRATION=1 to run manually")
	}
	runUOWTests(t, setupSQLServer(t))
}
