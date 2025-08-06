package borm

import (
	"reflect"
	"strings"
)

type DatabasesCache map[DatabaseName]*DatabaseRegistry
type RolesCache map[RoleName]RoleMethods
type TablesCache map[TableName]*TableRegistry
type TypesCache map[TypName]TypMethods

var databases = DatabasesCache{}
var roles = newRolesCache()

func newTableCache() *TablesCache {
	return &TablesCache{}
}
func newTypesCache() *TypesCache {
	return &TypesCache{}
}
func newRolesCache() *RolesCache {
	return &RolesCache{}
}
func (r *RolesCache) RegisterUser(name string, password string) *User {
	return RegisterUser(name, password)
}
func (r *TypesCache) RegisterEnum(name string, values ...any) *Enum {
	var stringValues []string
	for _, val := range values {
		typ := reflect.TypeOf(val)
		if typ.Kind() == reflect.String {
			v := reflect.ValueOf(val)
			stringValues = append(stringValues, v.String())
			continue
		}
		return &Enum{
			registerErrors: NewError("Values must be of the same kind of a string").Status(ErrInvalidType),
			Typ:            &Typ{Type: ENUM},
		}
	}
	typName := TypName(strings.ToLower(name))
	enum := &Enum{Typ: &Typ{Name: typName, Type: ENUM}, options: stringValues}
	(*r)[typName] = enum
	return enum
}
