package transaction

import (
	"database/sql"

	"github.com/Noeeekr/borm/errors"
	"github.com/Noeeekr/borm/internal/registers"
)

// Manager creates, starts and commits transactions
type Manager struct {
	database *sql.DB
}

func NewManager(db *sql.DB) *Manager {
	return &Manager{
		database: db,
	}
}

// Start starts a transaction on the manager and returns the transaction.. If another transaction is happening it returns the current transaction.
func (m *Manager) Start() (*Transaction, *errors.Error) {
	tx, err := m.database.Begin()
	if err != nil {
		return nil, errors.New(err.Error()).Status(errors.ErrFailedTransactionStart)
	}

	return NewTransaction(tx), nil
}

// No transaction
func (m *Manager) Do(query *registers.Query) *errors.Error {
	if query.Error != nil {
		return query.Error
	}

	stmt, err := m.database.Prepare(query.Query)
	if err != nil {
		return errors.New(err.Error()).
			Status(errors.ErrSyntax)
	}

	rows, err := stmt.Query(query.CurrentValues...)
	if err != nil {
		return errors.New("Failed operation. " + err.Error()).Status(errors.ErrFailedTransaction)
	}

	if query.RowsScanner != nil {
		return query.Scan(rows)
	}
	return nil
}
