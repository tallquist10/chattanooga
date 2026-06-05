package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

type PgSqlInfo struct {
	Host     string
	Port     int
	DbName   string
	DbUser   string
	Password string
}

func (pg *PgSqlInfo) ConnectionString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		pg.DbUser, pg.Password, pg.Host, pg.Port, pg.DbName)
}

func SqlMigrations() {
	adminPsqlInfo := &PgSqlInfo{
		Host:     os.Getenv("PGHOST"),
		Password: os.Getenv("PGPASSWORD"),
		DbName:   "postgres",
		DbUser:   "postgres",
		Port:     5432,
	}

	conn, err := sql.Open("postgres", adminPsqlInfo.ConnectionString())
	driver, err := postgres.WithInstance(conn, &postgres.Config{})
	if err != nil {
		fmt.Println(err.Error())
	}
	defer conn.Close()
	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", os.Getenv("MIGRATIONS_FILE_PATH")),
		"postgres", driver)
	if err != nil {
		fmt.Println(err.Error())
	}
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		panic(err.Error())
	}
	slog.Info("Migrations completed successfully")
}
