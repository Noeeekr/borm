package borm

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/lib/pq"

	"github.com/Noeeekr/borm/configuration"
)

const (
	DEBUGGING  = configuration.DEBUGGING
	PRODUCTION = configuration.PRODUCTION
)

func Settings() *configuration.Configuration {
	return configuration.Settings()
}
func Connect(registor *DatabaseRegistry) (*Commiter, error) {
	if registor.Host == "" || registor.Name == "" || registor.Owner.password == "" || registor.Owner.Name == "" {
		return nil, errors.New("cannot connect to database. trying to connect with broken (or empty) struct. ")
	}
	db, err := sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", registor.Owner.Name, registor.Owner.password, registor.Host, registor.Name))
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return newCommiter(registor, registor.Host, db), nil
}
