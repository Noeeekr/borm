package borm

import (
	"database/sql"

	"github.com/Noeeekr/borm/internal/migrations"
	"github.com/Noeeekr/borm/internal/registers"
)

func On(name registers.DatabaseName, owner registers.RoleName, db *sql.DB) *migrations.DatabaseManager {
	return migrations.NewDatabaseManager(name, owner, db)
}
