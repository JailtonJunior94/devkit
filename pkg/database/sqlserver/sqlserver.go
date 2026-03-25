package sqlserver

import (
	_ "github.com/microsoft/go-mssqldb"

	"devkit/pkg/database/internal/driverreg"
)

func init() {
	driverreg.Register("sqlserver")
}
