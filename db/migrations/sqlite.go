package migrations

import (
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/mattn/go-sqlite3"
)

func SqliteMigrations(file string) {
	db, err := sql.Open("sqlite3", file)
	if err != nil {
		slog.Error(err.Error())
	}

	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		slog.Error(err.Error())
	}

	defer db.Close()
	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", os.Getenv("MIGRATIONS_FILE_PATH")),
		"sqlite3", driver)
	if err != nil {
		slog.Error(err.Error())
	}
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		log.Fatal(err.Error())
	}
	slog.Info("Sqlite migrations completed successfully")
}
