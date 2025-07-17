package errors

import (
	"fmt"
	"runtime/debug"

	"github.com/Noeeekr/borm/configuration"
)

type Error struct {
	Stat       string
	Desc       string
	Joins      []*Error
	DebugStack string
}

type ErrorStatus string

const (
	ErrInvalidType               ErrorStatus = "Invalid type"
	ErrFound                     ErrorStatus = "Found"
	ErrNotFound                  ErrorStatus = "Not found"
	ErrEmpty                     ErrorStatus = "Empty"
	ErrSyntax                    ErrorStatus = "Syntax error"
	ErrInvalidMethodChain        ErrorStatus = "Invalid method chaining"
	ErrFailedOperation           ErrorStatus = "Failed operation"
	ErrFailedTransaction         ErrorStatus = "Failed transaction"
	ErrFailedTransactionStart    ErrorStatus = "Failed transaction start"
	ErrFailedTransactionCommit   ErrorStatus = "Failed transaction commit"
	ErrFailedTransactionRollback ErrorStatus = "Failed transaction rollback"
	ErrBadConnection             ErrorStatus = "Bad connection"
)

func New(description string) *Error {
	var debugStack string
	if configuration.Settings().Environment().GetEnvironment() == configuration.DEBUGGING {
		debugStack += "\n\n[Debugging Stack]: \n\n" + string(debug.Stack())
	}
	return &Error{
		Stat:       "",
		Desc:       description,
		DebugStack: debugStack,
	}
}

func (e *Error) String() string {
	var subjacentErrors string
	for i, err := range e.Joins {
		subjacentErrors += "\n"
		for range i {
			subjacentErrors += "\t"
		}
		subjacentErrors += err.String()
	}

	return e.Stat + e.Desc + subjacentErrors + e.DebugStack
}

// Appends to the end of the last description with a separator
func (e *Error) Append(d string) *Error {
	e.Desc += ": " + d
	return e
}

// Inserts before all other descriptions with a separator
func (e *Error) Insert(d string) *Error {
	e.Desc += d + ": "
	return e
}
func (e *Error) Status(s ErrorStatus) *Error {
	e.Stat = fmt.Sprintf("[%s]: ", s)
	return e
}

// Inserts a new error under the last error as a subjacent error
func (e *Error) Join(e2 *Error) *Error {
	if e2 == nil {
		return e
	}
	e.Joins = append(e.Joins, e2)
	return e
}
