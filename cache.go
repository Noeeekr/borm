package borm

import "strings"

type DatabasesCache map[DatabaseName]*DatabaseRegistor
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
func (r *TypesCache) RegisterEnum(name string, values ...string) *Enum {
	typName := TypName(strings.ToLower(name))
	enum := &Enum{Typ: &Typ{Name: typName, Type: ENUM}, options: values}
	(*r)[typName] = enum
	return enum
}
