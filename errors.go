package borm

import "fmt"

type Error struct {
	Stat ErrorStatus
	Desc string
}

type ErrorStatus string

const (
	ErrInvalidType        ErrorStatus = "Invalid type"
	ErrFound              ErrorStatus = "Found"
	ErrNotFound           ErrorStatus = "Not found"
	ErrEmpty              ErrorStatus = "Empty"
	ErrSyntax             ErrorStatus = "Syntax error"
	ErrInvalidMethodChain ErrorStatus = "Invalid method chaining"
)

func NewError() *Error {
	return &Error{
		Stat: "",
		Desc: "",
	}
}

func (e *Error) String() string {
	return fmt.Sprintf("[%s] %s", e.Stat, e.Desc)
}
func (e *Error) Append(d string) *Error {
	e.Desc += d
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
