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

func (t *Transaction) Do(query *Query) *Error {
	if err := query.validateFields(); err != nil {
		return err
	}
	if query == nil {
		return NewError("Failed transaction").Append("Unable to proceed, cannot use empty queries").Status(ErrSyntax)
	}
	if query.Error != nil {
		return query.Error
	}

	stmt, err := t.tx.Prepare(query.Query)
	if err != nil {
		err := NewError(err.Error()).Status(ErrSyntax)
		if Settings().Environment().GetEnvironment() == DEBUGGING {
			err.Append("\n[Query]: " + query.Query)
		}
		return err
	}

	if query.RowsScanner != nil {
		return t.query(stmt, query)
	}
	return t.exec(stmt, query.CurrentValues...)
}

func (t *Transaction) Commit() *Error {
	if err := t.tx.Commit(); err != nil {
		return NewError(err.Error()).Status(ErrFailedTransactionCommit)
	}

	return nil
}

func (t *Transaction) query(stmt *sql.Stmt, query *Query) *Error {
	rows, err := stmt.Query(query.CurrentValues...)
	if err != nil {
		return NewError(err.Error()).Join(t.rollback()).Status(ErrFailedTransaction)
	}

	if err := query.Scan(rows); t != nil {
		return err
	}

	return nil
}

func (t *Transaction) exec(stmt *sql.Stmt, args ...any) *Error {
	_, err := stmt.Exec(args...)
	if err != nil {
		return NewError("Transaction failed").Append(err.Error()).Join(t.rollback()).Status(ErrFailedTransaction)
	}
	return nil
}

func (t *Transaction) rollback() *Error {
	if err := t.tx.Rollback(); err != nil {
		return NewError("Failed rollback. ").Append(err.Error())
	}
	return nil
}

// ScannerFindOne is a scanner helper function. Returns true if at least one row is returned. Doesn't throw ErrNotFound. Instead returns false.
var ScannerFindOne = func(exists *bool) QueryRowsScanner {
	return func(rows *sql.Rows, throErrorOnFound bool) *Error {
		defer rows.Close()
		*exists = rows.Next()
		return nil
	}
}
