package transaction

import (
	"database/sql"

	"github.com/Noeeekr/borm/common"
	"github.com/Noeeekr/borm/internal/registers"
)

// Manager creates, starts and commits transactions
type Manager struct {
	currentTransaction *Transaction
	database           *sql.DB
}

func NewManager(db *sql.DB) *Manager {
	return &Manager{
		currentTransaction: nil,
		database:           db,
	}
}

// Start starts a transaction on the manager and returns the transaction.. If another transaction is happening it returns the current transaction.
func (m *Manager) Start() (*Transaction, *common.Error) {
	if m.currentTransaction != nil {
		return m.currentTransaction, nil
	}

	tx, err := m.database.Begin()
	if err != nil {
		return nil, common.NewError(err.Error()).Status(common.ErrFailedTransactionStart)
	}

	m.currentTransaction = NewTransaction(tx)
	return m.currentTransaction, nil
}
func (m *Manager) Commit() *common.Error {
	if m.currentTransaction == nil {
		return common.NewError("Unable to commit, no transaction in progress.").Status(common.ErrFailedTransactionCommit)
	}

	_, err := m.currentTransaction.Commit()
	m.currentTransaction = nil
	return err
}

func (m *Manager) Query(query *registers.Query) *common.Error {
	stmt, err := m.database.Prepare(query.Query)
	if err != nil {
		return common.NewError(err.Error()).
			Status(common.ErrSyntax)
	}

	rows, err := stmt.Query(query.CurrentValues...)
	if err != nil {
		return common.NewError("Failed operation. " + err.Error()).Status(common.ErrFailedTransaction)
	}

	if query.RowsScanner != nil {
		return query.Scan(rows)
	}
	return nil
}
