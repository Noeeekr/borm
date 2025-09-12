package borm

import (
	"database/sql"
	"errors"
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
func (m *TransactionFactory) StartTx() (*Transaction, error) {
	tx, err := m.database.Begin()
	if err != nil {
		return nil, errors.Join(ErrFailedTransaction, err)
	}

	return NewTransaction(tx), nil
}

// No transaction
func (m *TransactionFactory) Do(query *Query) error {
	if query.Error != nil {
		return query.Error
	}
	if err := query.isValid(); err != nil {
		return err
	}

	stmt, err := m.database.Prepare(query.build())
	if err != nil {
		return errors.Join(ErrSyntax, err)
	}

	rows, err := stmt.Query(query.CurrentValues...)
	if err != nil {
		return errors.Join(ErrFailedTransaction, err)
	}

	if query.RowsScanner != nil {
		found, err := query.Scan(rows)
		if err != nil {
			return err
		}
		// Found, Throw Error On Found
		if found && query.throwErrorOnFound {
			return ErrorDescription(ErrFound, "Rows found")
		}
		// Not Found, Default Throw Error On Not Found
		if !found && !query.throwErrorOnFound {
			return ErrorDescription(ErrNotFound, "No rows found")
		}
	}
	return nil
}
