package borm

import (
	"fmt"
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
	if len(values) == 0 {
		return &Enum{
			registerErrors: ErrInvalidType,
			Typ:            &Typ{Type: ENUM},
		}
	}

	// Validate value types
	options := []any{}
	targetType := reflect.TypeOf(values[0])
	for _, value := range values {
		// Values are not from the same type
		typ := reflect.TypeOf(value)
		if typ.Kind() != targetType.Kind() {
			fmt.Println(targetType.Kind(), typ.Kind())
			return &Enum{
				registerErrors: ErrInvalidType,
				Typ:            &Typ{Type: ENUM},
			}
		}

		// Values are not from valid kind
		if (typ.Kind() < 2 || typ.Kind() > 11) && typ.Kind() != 24 {
			return &Enum{
				registerErrors: ErrInvalidType,
				Typ:            &Typ{Type: ENUM},
			}
		}

		options = append(options, value)
	}
	typName := TypName(strings.ToLower(name))
	enum := &Enum{Typ: &Typ{Name: typName, Type: ENUM}, options: options, kind: targetType.Kind()}
	(*r)[typName] = enum
	return enum
}
