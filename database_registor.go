package borm

import (
	"strings"
)

type DatabaseName string
type DatabaseRegistor struct {
	Name  DatabaseName
	Owner *User

	*TablesCache
	*RolesCache
}

func (*DatabaseRegistor) RegisterDatabase(name string, owner *User) *DatabaseRegistor {
	databaseName := DatabaseName(strings.ToLower(name))
	if database, ok := Databases[databaseName]; ok {
		return database
	}
	database := newDatabaseRegistor(databaseName, owner)
	Databases[databaseName] = database
	return database
}
func newDatabaseRegistor(name DatabaseName, owner *User) *DatabaseRegistor {
	return &DatabaseRegistor{
		Name:        name,
		Owner:       owner,
		TablesCache: newTableCache(),
		RolesCache:  newRoleCache(),
	}
}
