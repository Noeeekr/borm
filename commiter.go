package borm

import (
	"database/sql"
)

type Commiter struct {
	host string
	db   *sql.DB

	// Tells if something was already created or not
	RegistorCache map[string]bool

	*RolesCache
	*DatabaseRegistry
	*TransactionFactory
	*MigrationPopulator
}

func (m *Commiter) DB() *sql.DB {
	return m.db
}

func newCommiter(r *DatabaseRegistry, host string, db *sql.DB) *Commiter {
	return &Commiter{
		host:               host,
		db:                 db,
		DatabaseRegistry:   r,
		RolesCache:         roles,
		RegistorCache:      map[string]bool{},
		TransactionFactory: newTransactionFactory(db),
		MigrationPopulator: newMigrationPopulator(),
	}
}
