package postgres

import (
	_ "github.com/lib/pq"

	"devkit/pkg/database/internal/driverreg"
)

func init() {
	driverreg.Register("postgres")
}
