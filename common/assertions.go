package common

import (
	"reflect"

	"github.com/Noeeekr/borm/errors"
)

func IsStruct(t reflect.Type) *errors.Error {
	if t.Kind() != reflect.TypeFor[struct{}]().Kind() {
		return errors.New(t.Name() + " must be of kind struct").Status(errors.ErrInvalidType)
	}
	return nil
}
