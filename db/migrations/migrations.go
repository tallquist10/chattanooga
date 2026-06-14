package migrations

import (
	"os"
)

func SqlMigrations() {
	sqlMode := os.Getenv("SQL_MODE")
	if sqlMode == "POSTGRES" {
		PgSqlMigrations()
		return
	}
	if sqlMode == "SQLITE" {
		SqliteMigrations(os.Getenv("SQLITE_FILE"))
		return
	}
	panic("Only POSTGRES and SQLITE are valid options for SQL_MODE")
}
