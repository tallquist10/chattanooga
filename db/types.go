package db

import "fmt"

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
