// Package nsactions contains types that abstract the golang database/sql package transaction operations for easy chaining, gracefull errors and operations.
//
//	Request
//
// Allows creating queries and making requests from those queries.
//
//	Transaction
//
// Allows making chained transactions with gracefull error handling.
package transaction

import (
	"database/sql"

	"github.com/Noeeekr/borm/common"
	"github.com/Noeeekr/borm/internal/registers"
)

// Transaction Automatically switches between Query() and Exec() when necessary.
//
// Transaction contains a common.Error that is different than nil if an error happened at any moment.
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

func (t *Transaction) Query(query *registers.Query) (*Transaction, *common.Error) {
	if query == nil {
		return t, common.NewError("Invalid query. Empty query.").Status(common.ErrSyntax)
	}

	stmt, err := t.tx.Prepare(query.Query)
	if err != nil {
		return t, common.NewError(err.Error()).Status(common.ErrSyntax)
	}

	if query.RowsScanner != nil {
		return t.query(stmt, query)
	}
	return t.exec(stmt, query.CurrentValues...)
}

func (t *Transaction) Commit() (*Transaction, *common.Error) {
	if err := t.tx.Commit(); err != nil {
		return t, common.NewError("Transaction Failed: " + err.Error()).Status(common.ErrFailedTransactionCommit)
	}

	return t, nil
}

func (t *Transaction) query(stmt *sql.Stmt, query *registers.Query) (*Transaction, *common.Error) {
	rows, err := stmt.Query(query.CurrentValues...)
	if err != nil {
		return nil, common.NewError(err.Error()).Join(t.rollback()).Status(common.ErrFailedTransaction)
	}

	if err := query.Scan(rows); t != nil {
		return t, err
	}

	return t, nil
}

func (t *Transaction) exec(stmt *sql.Stmt, args ...any) (*Transaction, *common.Error) {
	_, err := stmt.Exec(args...)
	if err != nil {
		return t, common.NewError("Transaction failed").Append(err.Error()).Join(t.rollback()).Status(common.ErrFailedTransaction)
	}
	return t, nil
}

func (t *Transaction) rollback() *common.Error {
	if err := t.tx.Rollback(); err != nil {
		return common.NewError("Unable to rollback. " + err.Error())
	}
	return nil
}

// CheckExist is a Scanner helper function that checks if at least one row exist and scans the result to a boolean.
var CheckExist = func(exists *bool) registers.QueryRowsScanner {
	return func(rows *sql.Rows, throwErrorOnFound bool) *common.Error {
		*exists = rows.Next()
		rows.Close()
		return nil
	}
}
