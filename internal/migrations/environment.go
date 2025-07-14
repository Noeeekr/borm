package migrations

import (
	"fmt"

	"github.com/Noeeekr/borm/common"
	"github.com/Noeeekr/borm/internal/registers"
)

func (m *DatabaseManager) Environment(database *registers.Database) (*DatabaseManager, *common.Error) {
	// Create the environment queries
	queries := []*registers.Query{}
	for _, role := range *m.Database.RoleCache {
		if role.RoleType() == registers.USER {
			user := role.(*registers.User)
			queries = append(queries, parseCreateUserQuery(user))
		}
	}

	for _, query := range queries {
		_, err := m.db.Exec(query.Query, query.CurrentValues...)
		if err != nil {
			return nil, common.NewError().Description(err.Error()).Status(common.ErrSyntax).Append(query.Query)
		}
	}

	query := parseCreateDatabaseQuery(database)
	_, err := m.db.Exec(query.Query, query.CurrentValues...)
	if err != nil {
		return nil, common.NewError().Description(err.Error()).Status(common.ErrSyntax).Append(query.Query)
	}

	// migrate and then connect to new database at the end
	return NewDatabaseManager(database.Name, database.Owner, nil), nil
}

func parseCreateDatabaseQuery(database *registers.Database) *registers.Query {
	q := registers.Query{}
	q.Query = fmt.Sprintf("CREATE DATABASE %s WITH OWNER = %s;", database.Name, database.Owner)
	return &q
}
func parseCreateUserQuery(user *registers.User) *registers.Query {
	q := registers.Query{}
	q.Query = fmt.Sprintf("CREATE USER %s\n\tWITH LOGIN\n\tPASSWORD '%s';", user.Name, user.Password())
	return &q
}
