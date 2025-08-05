package borm

import (
	"database/sql"
)

// TransactionFactory creates, starts and commits transactions
type TransactionFactory struct {
	database *sql.DB
}

func newTransactionFactory(db *sql.DB) *TransactionFactory {
	return &TransactionFactory{
		database: db,
	}
}

// Start starts a transaction on the manager and returns the transaction.. If another transaction is happening it returns the current transaction.
func (m *TransactionFactory) StartTx() (*Transaction, *Error) {
	tx, err := m.database.Begin()
	if err != nil {
		return nil, NewError(err.Error()).Status(ErrFailedTransactionStart)
	}

	return NewTransaction(tx), nil
}

// No transaction
func (m *TransactionFactory) Do(query *Query) *Error {
	if query.Error != nil {
		return query.Error
	}
	if err := query.validateFields(); err != nil {
		return err
	}

	stmt, err := m.database.Prepare(query.Query)
	if err != nil {
		return NewError(err.Error()).
			Status(ErrSyntax)
	}

	rows, err := stmt.Query(query.CurrentValues...)
	if err != nil {
		return NewError("Failed operation. " + err.Error()).Status(ErrFailedTransaction)
	}

	if query.RowsScanner != nil {
		return query.Scan(rows)
	}
	return nil
}
