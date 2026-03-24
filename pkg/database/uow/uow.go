// Package uow provides a Unit of Work implementation for coordinating multiple
// repository operations within a single database transaction.
//
// Usage:
//
//	u, _ := uow.New(db)
//	u.Register("users", func(tx *sql.Tx) any { return NewUserRepo(tx) })
//
//	err := u.Do(ctx, func(ctx context.Context) error {
//	    repo, _ := uow.GetRepository[*UserRepo](ctx, u, "users")
//	    return repo.Save(ctx, user)
//	})
//
// When Do is not called, repositories operate directly on the *sql.DB without
// a transaction. Each call to Do creates an independent transaction, making
// UnitOfWork safe for concurrent use.
package uow

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
)

// Option configures optional UnitOfWork settings. Use the With... functions to
// build options and pass them to New.
type Option func(*uowOptions)

// uowOptions holds the optional settings applied by Options.
type uowOptions struct {
	txOpts *sql.TxOptions
}

// WithTxOptions sets the transaction options (isolation level, read-only mode)
// used by every Do call. Default: nil, which uses the database's default
// isolation level in read-write mode.
func WithTxOptions(opts *sql.TxOptions) Option {
	return func(o *uowOptions) { o.txOpts = opts }
}

func applyDefaults(_ *uowOptions) {
	// Default txOpts is nil: driver default isolation level, read-write.
}

// RepositoryFactory creates a repository instance bound to the given transaction.
// The returned value is stored in the transaction context and retrieved via
// GetRepository.
type RepositoryFactory func(tx *sql.Tx) any

// UnitOfWork manages transactional operations over a *sql.DB.
// Each call to Do creates an independent transaction, making it safe for
// concurrent use. The factory registry is protected by a read-write mutex.
type UnitOfWork struct {
	db        *sql.DB
	txOpts    *sql.TxOptions
	factories map[string]RepositoryFactory
	mu        sync.RWMutex
}

// New creates a UnitOfWork from a *sql.DB.
// Returns ErrDBRequired if db is nil.
func New(db *sql.DB, opts ...Option) (*UnitOfWork, error) {
	if db == nil {
		return nil, ErrDBRequired
	}

	o := &uowOptions{}
	applyDefaults(o)
	for _, opt := range opts {
		opt(o)
	}

	return &UnitOfWork{
		db:        db,
		txOpts:    o.txOpts,
		factories: make(map[string]RepositoryFactory),
	}, nil
}

// Register adds a repository factory under the given name.
// The factory is called with the active *sql.Tx each time Do is invoked,
// creating a fresh repository instance bound to the transaction.
// Register is safe for concurrent use.
func (u *UnitOfWork) Register(name string, factory RepositoryFactory) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.factories[name] = factory
}

// Do executes fn within a transaction. On success it commits; on error or
// panic it rolls back. If fn panics, the rollback is attempted and the panic
// is re-propagated. If both fn and rollback return errors, they are joined
// via errors.Join.
//
// Each call to Do creates an independent transaction, so concurrent calls
// are safe and do not share state.
func (u *UnitOfWork) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := u.db.BeginTx(ctx, u.txOpts)
	if err != nil {
		return fmt.Errorf("uow: begin transaction: %w", err)
	}

	u.mu.RLock()
	repos := make(map[string]any, len(u.factories))
	for name, factory := range u.factories {
		repos[name] = factory(tx)
	}
	u.mu.RUnlock()

	txCtx := contextWithTx(ctx, tx, repos)

	// panicked tracks whether fn completed without panic.
	// The defer performs rollback on panic; explicit rollback handles error returns.
	panicked := true
	defer func() {
		if panicked {
			_ = tx.Rollback()
		}
	}()

	if err := fn(txCtx); err != nil {
		panicked = false
		if rbErr := tx.Rollback(); rbErr != nil {
			return errors.Join(err, fmt.Errorf("uow: rollback: %w", rbErr))
		}
		return err
	}

	panicked = false
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("uow: commit: %w", err)
	}
	return nil
}

// GetRepository retrieves the repository registered under name, instantiated
// with the active transaction. T is the expected concrete type of the repository.
//
// Returns ErrNoActiveTransaction if called outside a Do callback.
// Returns ErrRepositoryNotFound if name was not registered.
// Returns an error if the stored value cannot be asserted to T.
func GetRepository[T any](ctx context.Context, u *UnitOfWork, name string) (T, error) {
	state, ok := txStateFromContext(ctx)
	if !ok {
		var zero T
		return zero, ErrNoActiveTransaction
	}
	repo, ok := state.repos[name]
	if !ok {
		var zero T
		return zero, fmt.Errorf("%w: %s", ErrRepositoryNotFound, name)
	}
	typed, ok := repo.(T)
	if !ok {
		var zero T
		return zero, fmt.Errorf("uow: repository %q type mismatch", name)
	}
	return typed, nil
}
