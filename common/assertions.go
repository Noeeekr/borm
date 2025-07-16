package common

import "reflect"

func IsStruct(t reflect.Type) *Error {
	if t.Kind() != reflect.TypeFor[struct{}]().Kind() {
		return NewError(t.Name() + " must be of kind struct").Status(ErrInvalidType)
	}
	return nil
}
