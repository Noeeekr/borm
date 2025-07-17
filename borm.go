package borm

import (
	_ "github.com/lib/pq"

	"github.com/Noeeekr/borm/configuration"
	"github.com/Noeeekr/borm/errors"
	"github.com/Noeeekr/borm/internal/manager"
	"github.com/Noeeekr/borm/internal/registers"
)

func Settings() *configuration.Configuration {
	return configuration.Settings()
}

func Connect(user string, password string, host string, database registers.DatabaseName) (*manager.DatabaseManager, *errors.Error) {
	return manager.Connect(registers.NewUser(user, password), host, database)
}
