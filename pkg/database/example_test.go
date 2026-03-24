package database_test

import (
	"context"
	"fmt"

	// Import the driver of your choice via side-effect in your main.go:
	//   import _ "github.com/lib/pq"
	//   import _ "github.com/go-sql-driver/mysql"
	//   import _ "github.com/microsoft/go-mssqldb"

	"devkit/pkg/database"
)

// ExampleNew demonstrates Scenario A: simple queries with the Database Manager.
// The consumer obtains a *sql.DB from Manager.DB() and uses it directly.
func ExampleNew() {
	ctx := context.Background()

	mgr, err := database.New(ctx, database.Config{
		Driver: "postgres",
		DSN:    "postgres://user:pass@localhost/mydb?sslmode=disable",
	})
	if err != nil {
		fmt.Printf("failed to connect: %v\n", err)
		return
	}
	defer func() { _ = mgr.Close(ctx) }()

	// Use the native *sql.DB for direct queries.
	rows, err := mgr.DB().QueryContext(ctx, "SELECT id, name FROM users")
	if err != nil {
		fmt.Printf("query error: %v\n", err)
		return
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			fmt.Printf("scan error: %v\n", err)
			return
		}
		fmt.Printf("user: %d %s\n", id, name)
	}
}

// ExampleManager_Close demonstrates idempotent shutdown.
func ExampleManager_Close() {
	ctx := context.Background()

	mgr, err := database.New(ctx, database.Config{
		Driver: "postgres",
		DSN:    "postgres://user:pass@localhost/mydb?sslmode=disable",
	})
	if err != nil {
		fmt.Printf("failed to connect: %v\n", err)
		return
	}

	// Close may be called multiple times safely.
	_ = mgr.Close(ctx)
	_ = mgr.Close(ctx) // second call is a no-op
}
