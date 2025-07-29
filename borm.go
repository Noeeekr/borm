package borm

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"

	"github.com/Noeeekr/borm/configuration"
)

func Settings() *configuration.Configuration {
	return configuration.Settings()
}
func Connect(username string, password string, host string, dbname string) (*Commiter, *Error) {
	db, err := sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", username, password, host, dbname))
	if err != nil {
		return nil, NewError(err.Error()).Status(ErrBadConnection)
	}
	if err := db.Ping(); err != nil {
		return nil, NewError("Unable to ping database").Append(err.Error()).Status(ErrBadConnection)
	}
	return newCommiter(dbname, newUser(username, password), db, host), nil
}
