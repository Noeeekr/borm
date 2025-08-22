package borm

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrConfiguration error = errors.New("Configuration Necessary")

	ErrInvalidMethodChain error = errors.New("Invalid method chaining")
	ErrInvalidType        error = errors.New("Invalid type")
	ErrSyntax             error = errors.New("Syntax error")

	ErrNotFound error = errors.New("Not found")
	ErrFound    error = errors.New("Found")

	ErrFailedOperation           error = errors.New("Failed operation")
	ErrFailedTransaction         error = errors.New("Failed transaction")
	ErrFailedTransactionStart    error = errors.New("Failed transaction start")
	ErrFailedTransactionCommit   error = errors.New("Failed transaction commit")
	ErrFailedTransactionRollback error = errors.New("Failed transaction rollback")

	ErrBadConnection error = errors.New("Bad connection")
	ErrUnexpected    error = errors.New("Unexpected")
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
	return fmt.Errorf("%w\n%w", e1, e2)
}
