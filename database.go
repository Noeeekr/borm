package borm

import (
	"database/sql"

	"github.com/Noeeekr/borm/internal/registers"
)

type Database struct {
	Migrate *Migrate
	db      *sql.DB
}

func NewDatabase(db *sql.DB) *Database {
	return &Database{
		Migrate: NewMigrate(),
		db:      db,
	}
}
func (m *Database) Table(table any) *registers.Table {
	return registers.Tables.Table(table)
}
func (m *Database) Enum(name string, values ...string) registers.EnumMethods {
	return registers.Roles.Enum(name, values...)
}
func (m *Database) User(name string) *registers.User {
	return registers.Roles.User(name)
}
func (m *Database) Database(name, owner string) {

}
