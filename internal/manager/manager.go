package manager

import (
	"database/sql"
	"fmt"

	"github.com/Noeeekr/borm/common"
	"github.com/Noeeekr/borm/internal/registers"
)

type DatabaseManager struct {
	host string
	db   *sql.DB
	*registers.Database
}

func Connect(user, password, host, database string) (*DatabaseManager, *common.Error) {
	db, err := sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", user, password, host, database))
	if err != nil {
		return nil, common.NewError(err.Error()).Status(common.ErrBadConnection)
	}
	return New(registers.DatabaseName(database), registers.NewUser(user, password), db), nil
}
func New(name registers.DatabaseName, user *registers.User, db *sql.DB) *DatabaseManager {
	return &DatabaseManager{
		db:       db,
		Database: registers.NewDatabase(name, user),
	}
}
func (m *DatabaseManager) DB() *sql.DB {
	return m.db
}
func (m *DatabaseManager) NewDatabase(name string, owner *registers.User) *registers.Database {
	return registers.Databases.Database(name, owner)
}
