package borm

import (
	"strings"
)

type DatabaseName string
type DatabaseRegistry struct {
	Name  DatabaseName
	Host  string
	Owner *User

	*TablesCache
	*TypesCache
}

func (r *DatabaseRegistry) RegisterDatabase(dbname DatabaseName, owner *User) *DatabaseRegistry {
	return RegisterDatabase(string(dbname), r.Host, owner)
}
func RegisterDatabase(dbname string, host string, owner *User) *DatabaseRegistry {
	databaseName := DatabaseName(strings.ToLower(dbname))
	if database, ok := databases[databaseName]; ok {
		return database
	}
	return &DatabaseRegistry{
		Name:        databaseName,
		Owner:       owner,
		Host:        host,
		TablesCache: newTableCache(),
		TypesCache:  newTypesCache(),
	}
}
