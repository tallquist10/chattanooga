package migrations

import (
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/tallquist10/chat-server/db"
)

func PgSqlMigrations() {
	adminPsqlInfo := &db.PgSqlInfo{
		Host:     os.Getenv("PGHOST"),
		Password: os.Getenv("PGPASSWORD"),
		DbName:   "postgres",
		DbUser:   "postgres",
		Port:     5432,
	}

	conn, err := sql.Open("postgres", adminPsqlInfo.ConnectionString())
	driver, err := postgres.WithInstance(conn, &postgres.Config{})
	if err != nil {
		slog.Error(err.Error())
	}
	defer conn.Close()
	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", os.Getenv("MIGRATIONS_FILE_PATH")),
		"postgres", driver)
	if err != nil {
		slog.Error(err.Error())
	}
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		log.Fatal(err.Error())
	}
	slog.Info("Postgres migrations completed successfully")
}
