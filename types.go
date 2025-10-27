package borm

import "reflect"

type TypName string
type TypType string
type TypMethods interface {
	GetName() TypName
	GetType() TypType
}
type Typ struct {
	TypMethods
	Name TypName
	Type TypType
}

type EnumMethods interface {
	GetValues() []string
}
type Enum struct {
	EnumMethods
	*Typ

	kind           reflect.Kind
	registerErrors error
	options        []any
}

const (
	ENUM TypType = "enum"
)

func (t *Typ) GetName() TypName {
	return t.Name
}
func (t *Typ) GetType() TypType {
	return t.Type
}
func (e *Enum) GetValues() []any {
	return e.options
}
