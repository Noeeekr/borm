package manager

import (
	"fmt"

	"github.com/Noeeekr/borm/common"
	"github.com/Noeeekr/borm/internal/registers"
)

func (m *DatabaseManager) Environment(database *registers.Database, configuration *Configuration) (*DatabaseManager, *common.Error) {
	if configuration == nil {
		configuration = &Configuration{}
	}

	var err *common.Error
	createdUsers := []*registers.User{}
	for _, role := range *m.Database.RoleCache {
		if role.RoleType() != registers.USER {
			continue
		}
		user := role.(*registers.User)

		fmt.Println("[Creating database user]:", user.Name)
		err = m.createDatabaseUser(configuration, user)
		if err != nil {
			fmt.Printf("[%s]: Failure on database user creation: %s\n", err.Stat, err.Desc)
			break
		}
		createdUsers = append(createdUsers, user)
	}
	if err != nil {
		if configuration.undoOnError {
			return nil, err.Join(m.dropDatabaseUsers(createdUsers...))
		}
		return nil, err
	}

	_, err2 := m.db.Exec(parseCreateDatabaseQuery(database).Query)
	if err2 != nil {
		return nil, common.NewError().Description(err2.Error()).Status(common.ErrSyntax)
	}

	return Connect(string(database.Owner.Name), database.Owner.Password(), m.host, string(database.Name))
}

func (m *DatabaseManager) dropDatabaseUsers(users ...*registers.User) *common.Error {
	for _, user := range users {
		rows, err := m.db.Query("SELECT datname FROM pg_catalog.pg_database d INNER JOIN pg_catalog.pg_roles u ON d.datdba = u.oid WHERE rolname = $1;", user.Name)
		if err != nil {
			return common.NewError().Description(err.Error()).Status(common.ErrFailedOperation)
		}

		var datnames []string
		for rows.Next() {
			var datname string
			err := rows.Scan(&datname)
			if err != nil {
				return common.NewError().Description(err.Error()).Status(common.ErrFailedOperation)
			}
			datnames = append(datnames, datname)
		}
		rows.Close()

		for _, datname := range datnames {
			_, err := m.db.Exec("DROP DATABASE " + datname)
			if err != nil {
				return common.NewError().Description("Failure while dropping database").After(err.Error()).Status(common.ErrFailedOperation)
			}
		}

		dropUserQuery := parseDropUserQuery(user)
		_, err = m.db.Exec(dropUserQuery.Query)
		if err != nil {
			return common.NewError().Description(err.Error()).Status(common.ErrFailedOperation)
		}
	}
	return nil
}
func (m *DatabaseManager) createDatabaseUser(configuration *Configuration, user *registers.User) *common.Error {
	rows, err := m.db.Query("SELECT rolname FROM pg_catalog.pg_roles WHERE rolname = $1;", user.Name)
	if err != nil {
		return common.NewError().Description("Failure while checking if user exists").After(err.Error()).Status(common.ErrSyntax)
	}

	var userExists bool
	if rows.Next() {
		if configuration.ignoreExisting {
			return nil
		}
		userExists = true
	}
	rows.Close()

	if userExists && configuration.reacreateExisting {
		err := m.dropDatabaseUsers(user)
		if err != nil {
			return err
		}
	}

	createUserQuery := parseCreateUserQuery(user)
	_, err = m.db.Exec(createUserQuery.Query)
	if err != nil {
		return common.NewError().Description("Failed creating user").After(err.Error()).Status(common.ErrFailedOperation)
	}
	return nil
}
func parseCreateDatabaseQuery(database *registers.Database) *registers.Query {
	q := registers.Query{}
	q.Query = fmt.Sprintf("CREATE DATABASE %s WITH OWNER = %s;", database.Name, database.Owner.Name)
	return &q
}
func parseCreateUserQuery(user *registers.User) *registers.Query {
	q := registers.Query{}
	q.Query = fmt.Sprintf("CREATE USER %s\n\tWITH LOGIN\n\tPASSWORD '%s';", user.Name, user.Password())
	return &q
}
func parseDropUserQuery(user *registers.User) *registers.Query {
	q := registers.Query{}
	q.Query = fmt.Sprintf("DROP USER %s;", user.Name)
	return &q
}
