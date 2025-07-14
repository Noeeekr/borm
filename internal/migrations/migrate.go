package migrations

import (
	"database/sql"

	"github.com/Noeeekr/borm/internal/registers"
)

type DatabaseManager struct {
	db *sql.DB
	*registers.Database
}

func NewDatabaseManager(name registers.DatabaseName, owner registers.RoleName, db *sql.DB) *DatabaseManager {
	return &DatabaseManager{
		db:       db,
		Database: registers.NewDatabase(name, owner),
	}
}
func (m *DatabaseManager) NewDatabase(name string, owner registers.RoleName) *registers.Database {
	return registers.Databases.Database(name, owner)
}
