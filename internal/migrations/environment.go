package migrations

import (
	"github.com/Noeeekr/borm/common"
	"github.com/Noeeekr/borm/internal/registers"
)

func (m *DatabaseManager) Environment(database *registers.Database) (*DatabaseManager, *common.Error) {
	// Create the environment queries
	queries := []*registers.Query{}
	for _, role := range *database.RoleCache {
		if role.RoleType() == registers.USER {
			user := role.(*registers.User)
			queries = append(queries, parseCreateUserQuery(user))
		}
	}

	queries = append(queries, parseCreateDatabaseQuery(database))

	for _, query := range queries {
		_, err := m.db.Exec(query.Query, query.CurrentValues...)
		if err != nil {
			return nil, common.NewError().Description(err.Error()).Status(common.ErrSyntax)
		}
	}

	// migrate and then connect to new database at the end
	return NewDatabaseManager(database.Name, database.Owner, nil), nil
}

func parseCreateDatabaseQuery(database *registers.Database) *registers.Query {
	q := registers.Query{}
	q.Query = "CREATE DATABASE $1 WITH OWNER $2;"
	q.Values(database.Name, database.Owner)
	return &q
}
func parseCreateUserQuery(user *registers.User) *registers.Query {
	q := registers.Query{}
	q.Query = "CREATE USER $1\n\tWITH LOGIN\n\tPASSWORD $2;"
	q.Values(user.Name, user.Password())
	return &q
}
