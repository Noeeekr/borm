package manager

import (
	"database/sql"
	"fmt"

	"github.com/Noeeekr/borm/errors"
	"github.com/Noeeekr/borm/internal/registers"
	"github.com/Noeeekr/borm/internal/transaction"
)

type DatabaseManager struct {
	host string
	db   *sql.DB
	// Stores the created stuff
	cache map[string]bool

	*transaction.Manager
	Register *registers.Database
}

func Connect(user *registers.User, host string, database registers.DatabaseName) (*DatabaseManager, *errors.Error) {
	db, err := sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", user.Name, user.Password(), host, database))
	if err != nil {
		return nil, errors.New(err.Error()).Status(errors.ErrBadConnection)
	}
	if err := db.Ping(); err != nil {
		return nil, errors.New("Unable to ping database").Append(err.Error()).Status(errors.ErrBadConnection)
	}
	return New(registers.DatabaseName(database), user, db, host), nil
}

func New(name registers.DatabaseName, user *registers.User, db *sql.DB, host string) *DatabaseManager {
	return &DatabaseManager{
		host:     host,
		db:       db,
		Register: registers.NewDatabase(name, user),
		cache:    map[string]bool{},
		Manager:  transaction.NewManager(db),
	}
}
func (m *DatabaseManager) DB() *sql.DB {
	return m.db
}
func (m *DatabaseManager) NewDatabase(name string, owner *registers.User) *registers.Database {
	return registers.Databases.Database(name, owner)
}
