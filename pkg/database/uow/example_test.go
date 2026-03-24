package uow_test

import (
	"context"
	"database/sql"
	"fmt"

	"devkit/pkg/database/uow"
)

// UserRepository is a hypothetical repository that operates on a Querier,
// working transparently with or without an active transaction.
type UserRepository struct {
	q uow.Querier
}

func NewUserRepository(q uow.Querier) *UserRepository {
	return &UserRepository{q: q}
}

func (r *UserRepository) Save(ctx context.Context, name string) error {
	_, err := r.q.ExecContext(ctx, "INSERT INTO users (name) VALUES ($1)", name)
	return err
}

// ExampleUnitOfWork_Do demonstrates Scenario B: transactional operations
// with Unit of Work. Multiple repositories participate in the same transaction.
func ExampleUnitOfWork_Do() {
	var db *sql.DB // obtained from database.Manager.DB() or created directly

	u, err := uow.New(db)
	if err != nil {
		fmt.Printf("uow.New: %v\n", err)
		return
	}

	// Register a factory that creates UserRepository bound to the active tx.
	u.Register("users", func(tx *sql.Tx) any {
		return NewUserRepository(tx)
	})

	ctx := context.Background()
	err = u.Do(ctx, func(ctx context.Context) error {
		repo, err := uow.GetRepository[*UserRepository](ctx, u, "users")
		if err != nil {
			return err
		}
		if err := repo.Save(ctx, "Alice"); err != nil {
			return err
		}
		return repo.Save(ctx, "Bob")
		// On success: auto-commit. On error: auto-rollback.
	})
	if err != nil {
		fmt.Printf("transaction failed: %v\n", err)
		return
	}
	fmt.Println("users saved")
}

// ExampleGetRepository demonstrates retrieving a typed repository inside Do.
func ExampleGetRepository() {
	var db *sql.DB

	u, _ := uow.New(db)
	u.Register("users", func(tx *sql.Tx) any {
		return NewUserRepository(tx)
	})

	ctx := context.Background()
	_ = u.Do(ctx, func(ctx context.Context) error {
		repo, err := uow.GetRepository[*UserRepository](ctx, u, "users")
		if err != nil {
			return err
		}
		return repo.Save(ctx, "Charlie")
	})
}
