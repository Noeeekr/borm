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
	*DatabaseRegistor
	*TransactionFactory
}

/*
	Registor
		-> Registers users => Through [PUBLIC RegisterUser METHOD] from [TYPE USER REGISTOR] (Returns *User)
		-> Registers database => Through [PUBLIC RegisterDatabase METHOD] from [TYPE DATABASE REGISTOR] (Returns *Database)
			[HAS] -> Table and role cache [X Registor cache] that it passes to registers method that need cache.
			[HAS] -> Registers tables => Through [PUBLIC RegisterTable METHOD] from [TABLE REGISTOR] (Returns *Table)
			[HAS] -> Registers roles => Through [PUBLIC] Role Registor
*/

func (m *Commiter) DB() *sql.DB {
	return m.db
}

func newCommiter(r *DatabaseRegistor, host string, db *sql.DB) *Commiter {
	return &Commiter{
		host:               host,
		db:                 db,
		DatabaseRegistor:   r,
		RolesCache:         roles,
		RegistorCache:      map[string]bool{},
		TransactionFactory: newTransactionFactory(db),
	}
}
