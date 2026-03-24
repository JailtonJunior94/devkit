package uow

import (
	"context"
	"database/sql"
)

// Querier abstracts *sql.DB and *sql.Tx for repository use.
// Both *sql.DB and *sql.Tx satisfy this interface, allowing repositories to
// operate transparently with or without an active transaction.
type Querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}

type ctxKey struct{}

type txState struct {
	tx    *sql.Tx
	repos map[string]any
}

func contextWithTx(ctx context.Context, tx *sql.Tx, repos map[string]any) context.Context {
	return context.WithValue(ctx, ctxKey{}, &txState{tx: tx, repos: repos})
}

func txStateFromContext(ctx context.Context) (*txState, bool) {
	state, ok := ctx.Value(ctxKey{}).(*txState)
	return state, ok
}

// TxFromContext returns the active *sql.Tx from ctx, or nil if no transaction
// is present. It is intended for use inside a Do callback when direct access
// to the transaction is needed.
func TxFromContext(ctx context.Context) *sql.Tx {
	state, ok := txStateFromContext(ctx)
	if !ok {
		return nil
	}
	return state.tx
}
