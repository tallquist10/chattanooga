package db

import (
	"database/sql"
	"errors"
	"log/slog"
	"os"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func SqlLite(file string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", file)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func Postgres() (*sql.DB, error) {
	appPsqlInfo := &PgSqlInfo{
		Host:     os.Getenv("PGHOST"),
		Password: os.Getenv("PGPASSWORD"),
		DbName:   "postgres",
		DbUser:   "chatserver",
		Port:     5432,
	}
	db, err := sql.Open("postgres", appPsqlInfo.ConnectionString())
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}
	return db, nil
}

func New() (*sql.DB, error) {
	sqlMode := os.Getenv("SQL_MODE")
	if sqlMode == "POSTGRES" {
		return Postgres()
	}
	if sqlMode == "SQLITE" {
		return SqlLite(os.Getenv("SQLITE_FILE"))
	}
	return nil, errors.New("Only POSTGRES and SQLITE are valid options for SQL_MODE")
}
