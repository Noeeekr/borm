package registers

import (
	"strings"
)

type DatabasesCache map[DatabaseName]*Database
type DatabaseName string
type Database struct {
	Name  DatabaseName
	Owner *User

	*TableCache
	*RoleCache
}

var Databases = DatabasesCache{}

func NewDatabase(name DatabaseName, owner *User) *Database {
	return &Database{
		Name:       name,
		Owner:      owner,
		TableCache: NewTableCache(),
		RoleCache:  NewRoleCache(),
	}
}

func (c *DatabasesCache) Database(name string, owner *User) *Database {
	databaseName := DatabaseName(strings.ToLower(name))
	if database, ok := (*c)[databaseName]; ok {
		return database
	}
	database := NewDatabase(databaseName, owner)
	(*c)[databaseName] = database
	return database
}
