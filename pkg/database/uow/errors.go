package uow

import "errors"

var (
	// ErrDBRequired is returned when a nil *sql.DB is passed to New.
	ErrDBRequired = errors.New("uow: db is required")

	// ErrRepositoryNotFound is returned when GetRepository cannot find the named repository.
	ErrRepositoryNotFound = errors.New("uow: repository not found")

	// ErrNoActiveTransaction is returned when GetRepository or TxFromContext is
	// called outside of a Do call.
	ErrNoActiveTransaction = errors.New("uow: no active transaction in context")
)
