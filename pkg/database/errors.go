package database

import "errors"

var (
	// ErrDriverRequired is returned when the driver field is empty.
	ErrDriverRequired = errors.New("database: driver is required")

	// ErrDSNRequired is returned when the DSN field is empty.
	ErrDSNRequired = errors.New("database: dsn is required")

	// ErrUnsupportedDriver is returned when the driver is not in the supported list.
	ErrUnsupportedDriver = errors.New("database: unsupported driver")

	// ErrInvalidPoolConfig is returned when the pool configuration is invalid.
	ErrInvalidPoolConfig = errors.New("database: invalid pool configuration")
)
