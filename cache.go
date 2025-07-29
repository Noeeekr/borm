package borm

import "strings"

type DatabasesCache map[DatabaseName]*DatabaseRegistor
type RolesCache map[RoleName]RoleMethods
type TablesCache map[TableName]*TableRegistor

var Databases = DatabasesCache{}

func newTableCache() *TablesCache {
	return &TablesCache{}
}
func (r *RolesCache) RegisterUser(name, password string) *User {
	user := newUser(name, password)
	(*r)[user.Name] = user
	return user
}
func (r *RolesCache) RegisterEnum(name string, values ...string) *Enum {
	roleName := RoleName(strings.ToLower(name))
	enum := &Enum{Role: &Role{Name: roleName, Type: ENUM}, options: values}
	(*r)[roleName] = enum
	return enum
}
func newRoleCache() *RolesCache {
	return &RolesCache{}
}
