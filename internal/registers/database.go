package registers

import (
	"strings"
)

type DatabasesCache map[DatabaseName]*Database
type DatabaseName string
type Database struct {
	Name  DatabaseName
	Owner RoleName

	*TableCache
	*RoleCache
}

var Databases = DatabasesCache{}

func NewDatabase(name DatabaseName, owner RoleName) *Database {
	return &Database{
		Name:       name,
		Owner:      owner,
		TableCache: NewTableCache(),
		RoleCache:  NewRoleCache(),
	}
}

func (c *DatabasesCache) Database(name string, owner RoleName) *Database {
	databaseName := DatabaseName(strings.ToLower(name))
	if database, ok := (*c)[databaseName]; ok {
		return database
	}
	owner = RoleName(strings.ToLower(string(owner)))
	database := NewDatabase(databaseName, owner)
	(*c)[databaseName] = database
	return database
}
