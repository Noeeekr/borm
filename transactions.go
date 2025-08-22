// Package nsactions contains types that abstract the golang database/sql package transaction operations for easy chaining, gracefull  and operations.
//
//	Request
//
// Allows creating queries and making requests from those queries.
//
//	Transaction
//
// Allows making chained transactions with gracefull error handling.
package borm

import (
	"database/sql"
	"errors"
)

// Transaction Automatically switches between Query() and Exec() when necessary.
//
// Transaction contains a Error that is different than nil if an error happened at any moment.
//
// Methods on Transaction created with a nil pointer will commit at the end of operation.
// Methods on Transaction created with an already started transactions won't commit at the end of operation and will execute in the transaction.
type Transaction struct {
	tx *sql.Tx
}

// NewTransaction creates a transaction. If a tx is != nil all operations will be done in its context and won't commit at the end.
func NewTransaction(tx *sql.Tx) *Transaction {
	return &Transaction{
		tx: tx,
	}
}

func (t *Transaction) Do(query *Query) error {
	if err := query.validateFields(); err != nil {
		return err
	}
	if query == nil {
		return ErrorDescription(ErrSyntax, "Failed operation, cannot use empty queries")
	}
	if query.Error != nil {
		return query.Error
	}

	stmt, err := t.tx.Prepare(query.Query)
	if err != nil {
		err := ErrorJoin(ErrSyntax, err)
		if Settings().Environment().GetEnvironment() == DEBUGGING {
			ErrorJoin(err, errors.New("\n[Query]: "+query.Query))
		}
		return err
	}

	if query.RowsScanner != nil {
		return t.query(stmt, query)
	}
	return t.exec(stmt, query.CurrentValues...)
}

func (t *Transaction) Commit() error {
	if err := t.tx.Commit(); err != nil {
		return ErrorDescription(ErrFailedTransactionCommit, err.Error())
	}

	return nil
}

func (t *Transaction) query(stmt *sql.Stmt, query *Query) error {
	rows, err := stmt.Query(query.CurrentValues...)
	if err != nil {
		return ErrorJoin(ErrorDescription(ErrFailedTransaction, err.Error()), t.rollback())
	}

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

	return nil
}

func (t *Transaction) exec(stmt *sql.Stmt, args ...any) error {
	_, err := stmt.Exec(args...)
	if err != nil {
		return ErrorJoin(ErrorDescription(ErrFailedTransaction, "Transaction failed", err.Error()), t.rollback())
	}
	return nil
}

func (t *Transaction) rollback() error {
	if err := t.tx.Rollback(); err != nil {
		return ErrorDescription(ErrFailedTransactionRollback, err.Error())
	}
	return nil
}

// ScannerFindOne is a scanner helper function. Returns true if at least one row is returned. Doesn't throw ErrNotFound or ErrFound. Instead returns false.
var ScannerFindOne = func(exists *bool) ReturnScanner {
	return func(rows *sql.Rows) (bool, error) {
		defer rows.Close()
		*exists = rows.Next()
		return true, nil
	}
}
