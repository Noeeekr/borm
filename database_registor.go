package borm

import (
	"strings"
)

type DatabaseName string
type DatabaseRegistor struct {
	Name  DatabaseName
	Host  string
	Owner *User

	*TablesCache
	*TypesCache
}

func (r *DatabaseRegistor) RegisterDatabase(dbname DatabaseName, owner *User) *DatabaseRegistor {
	return RegisterDatabase(string(dbname), r.Host, owner)
}
func RegisterDatabase(dbname string, host string, owner *User) *DatabaseRegistor {
	databaseName := DatabaseName(strings.ToLower(dbname))
	if database, ok := databases[databaseName]; ok {
		return database
	}
	return &DatabaseRegistor{
		Name:        databaseName,
		Owner:       owner,
		Host:        host,
		TablesCache: newTableCache(),
		TypesCache:  newTypesCache(),
	}
}
