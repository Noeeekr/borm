package borm

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrConfiguration error = errors.New("configuration necessary")

	ErrInvalidMethodChain error = errors.New("invalid method chaining")
	ErrInvalidType        error = errors.New("invalid type")
	ErrSyntax             error = errors.New("syntax error")

	ErrNotFound error = errors.New("not found")
	ErrFound    error = errors.New("found")

	ErrFailedOperation           error = errors.New("failed operation")
	ErrFailedTransaction         error = errors.New("failed transaction")
	ErrFailedTransactionStart    error = errors.New("failed transaction start")
	ErrFailedTransactionCommit   error = errors.New("failed transaction commit")
	ErrFailedTransactionRollback error = errors.New("failed transaction rollback")

	ErrBadConnection error = errors.New("bad connection")
	ErrUnexpected    error = errors.New("unexpected")
)

func ErrorDescription(err error, messages ...string) error {
	description := strings.Builder{}
	for _, message := range messages {
		description.WriteString(": ")
		description.WriteString(message)
	}
	return fmt.Errorf("[%w]%s", err, description.String())
}
func ErrorJoin(e1, e2 error) error {
	return fmt.Errorf("%w: %w", e1, e2)
}
