package manager

import (
	"fmt"

	"github.com/Noeeekr/borm/configuration"
	"github.com/Noeeekr/borm/errors"
	"github.com/Noeeekr/borm/internal/registers"
)

func (m *DatabaseManager) Environment(database *registers.Database) (*DatabaseManager, *errors.Error) {
	configuration := configuration.Settings().Migrations()
	if !configuration.Enabled {
		return Connect(database.Owner, m.host, database.Name)
	}

	registeredUsers := []*registers.User{}
	for _, role := range *database.RoleCache {
		if role.RoleType() == registers.USER {
			user := role.(*registers.User)
			registeredUsers = append(registeredUsers, user)
		}
	}

	created, err := m.migrateUsers(registeredUsers...)
	if err != nil {
		if configuration.Undo {
			return nil, err.Join(m.dropDatabaseUsers(created...))
		}
		return nil, err
	}

	err = m.migrateDatabase(database)
	if err != nil {
		if configuration.Undo {
			return nil, err.Join(m.dropDatabaseUsers(created...))
		}
		return nil, err
	}

	return Connect(database.Owner, m.host, database.Name)
}

func (m *DatabaseManager) migrateUsers(users ...*registers.User) ([]*registers.User, *errors.Error) {
	created := []*registers.User{}
	for _, user := range users {
		fmt.Println("[Creating database user]:", user.Name)
		err := m.migrateDatabaseUser(user)
		if err != nil {
			return created, err
		}
		created = append(created, user)
	}
	return created, nil
}
func (m *DatabaseManager) migrateDatabaseUser(user *registers.User) *errors.Error {
	rows, err := m.db.Query("SELECT rolname FROM pg_catalog.pg_roles WHERE rolname = $1;", user.Name)
	if err != nil {
		return errors.New("Failure while checking if user exists").Append(err.Error()).Status(errors.ErrSyntax)
	}

	configuration := configuration.Settings().Migrations()

	var userExists bool
	if rows.Next() {
		if configuration.Ignore {
			return nil
		}
		userExists = true
	}
	rows.Close()

	if userExists && configuration.Recreate {

		err := m.dropDatabaseUsers(user)
		if err != nil {
			return err
		}
	}

	createUserQuery := parseCreateUserQuery(user)
	_, err = m.db.Exec(createUserQuery.Query)
	if err != nil {
		return errors.New("Failed creating user").Append(err.Error()).Status(errors.ErrFailedOperation)
	}
	return nil
}
func (m *DatabaseManager) migrateDatabase(database *registers.Database) *errors.Error {
	rows, err := m.db.Query("SELECT datname FROM pg_catalog.pg_database WHERE datname = $1;", database.Name)
	if err != nil {
		return errors.New(err.Error()).Status(errors.ErrSyntax)
	}

	configuration := configuration.Settings().Migrations()

	var exists bool = rows.Next()
	defer rows.Close()
	if exists && configuration.Ignore {
		return nil
	}
	if exists && configuration.Recreate {
		if err := m.dropDatabase(database); err != nil {
			return err
		}
	}

	_, err = m.db.Exec(parseCreateDatabaseQuery(database).Query)
	if err != nil {
		return errors.New(err.Error()).Status(errors.ErrFailedOperation)
	}
	return nil
}
func (m *DatabaseManager) dropDatabaseUsers(users ...*registers.User) *errors.Error {
	for _, user := range users {
		rows, err := m.db.Query("SELECT datname FROM pg_catalog.pg_database d INNER JOIN pg_catalog.pg_roles u ON d.datdba = u.oid WHERE rolname = $1;", user.Name)
		if err != nil {
			return errors.New(err.Error()).Status(errors.ErrFailedOperation)
		}

		var datnames []string
		for rows.Next() {
			var datname string
			err := rows.Scan(&datname)
			if err != nil {
				return errors.New(err.Error()).Status(errors.ErrFailedOperation)
			}
			datnames = append(datnames, datname)
		}
		rows.Close()

		for _, datname := range datnames {
			_, err := m.db.Exec("DROP DATABASE " + datname)
			if err != nil {
				return errors.New("Failure while dropping database").Append(err.Error()).Status(errors.ErrFailedOperation)
			}
		}

		dropUserQuery := parseDropUserQuery(user)
		_, err = m.db.Exec(dropUserQuery.Query)
		if err != nil {
			return errors.New(err.Error()).Status(errors.ErrFailedOperation)
		}
	}
	return nil
}
func (m *DatabaseManager) dropDatabase(database *registers.Database) *errors.Error {
	_, err := m.db.Exec(fmt.Sprintf("DROP DATABASE %s;", database.Name))
	if err != nil {
		return errors.New(err.Error()).Status(errors.ErrFailedOperation)
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
