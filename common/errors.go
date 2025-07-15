package common

import "fmt"

type Error struct {
	Stat ErrorStatus
	Desc string
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

func NewError() *Error {
	return &Error{
		Stat: "",
		Desc: "",
	}
}

func (e *Error) String() string {
	return fmt.Sprintf("[%s]: %s", e.Stat, e.Desc)
}
func (e *Error) After(d string) *Error {
	e.Desc += ": " + d
	return e
}
func (e *Error) Before(d string) *Error {
	e.Desc += d + ": "
	return e
}
func (e *Error) Description(d string) *Error {
	e.Desc = d
	return e
}
func (e *Error) Status(s ErrorStatus) *Error {
	e.Stat = s
	return e
}
func (e *Error) Join(e2 *Error) *Error {
	if e2 == nil {
		return e
	}
	e.Desc += "\n\t" + e2.String()
	return e
}
