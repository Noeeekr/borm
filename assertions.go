package borm

import "reflect"

func isStruct(t reflect.Type) *Error {
	if t.Kind() != reflect.TypeFor[struct{}]().Kind() {
		return NewError().Status(ErrInvalidType).Description(t.Name() + " must be of kind struct")
	}
	return nil
}
