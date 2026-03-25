package mysql

import (
	_ "github.com/go-sql-driver/mysql"

	"devkit/pkg/database/internal/driverreg"
)

func init() {
	driverreg.Register("mysql")
}
