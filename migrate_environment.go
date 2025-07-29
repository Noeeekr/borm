package borm

import (
	"fmt"

	"github.com/Noeeekr/borm/configuration"
)

// Environment migrates the environment if migrations is enabled and then attempts to connect to the database. If migrations is not enabled it jumps to the end.
func (m *Commiter) Environment(database *DatabaseRegistor) (*Commiter, *Error) {
	configuration := configuration.Settings().Migrations()
	if !configuration.Enabled {
		return Connect(string(database.Owner.Name), database.Owner.Password(), m.host, string(database.Name))
	}

	registeredUsers := []*User{}
	for _, role := range *database.RolesCache {
		if role.GetType() == USER {
			user := role.(*User)
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

	return Connect(string(database.Owner.Name), database.Owner.password, m.host, string(database.Name))
}

func (m *Commiter) migrateUsers(users ...*User) ([]*User, *Error) {
	created := []*User{}
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
func (m *Commiter) migrateDatabaseUser(user *User) *Error {
	rows, err := m.db.Query("SELECT rolname FROM pg_catalog.pg_roles WHERE rolname = $1;", user.Name)
	if err != nil {
		return NewError("Failure while checking if user exists").Append(err.Error()).Status(ErrSyntax)
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

	createUserQuery := m.parseCreateUserQuery(user)
	_, err = m.db.Exec(createUserQuery.Query)
	if err != nil {
		return NewError("Failed creating user").Append(err.Error()).Status(ErrFailedOperation)
	}
	return nil
}
func (m *Commiter) migrateDatabase(database *DatabaseRegistor) *Error {
	rows, err := m.db.Query("SELECT datname FROM pg_catalog.pg_database WHERE datname = $1;", database.Name)
	if err != nil {
		return NewError(err.Error()).Status(ErrSyntax)
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

	_, err = m.db.Exec(m.parseCreateDatabaseQuery(database).Query)
	if err != nil {
		return NewError(err.Error()).Status(ErrFailedOperation)
	}
	return nil
}
func (m *Commiter) dropDatabaseUsers(users ...*User) *Error {
	for _, user := range users {
		rows, err := m.db.Query("SELECT datname FROM pg_catalog.pg_database d INNER JOIN pg_catalog.pg_roles u ON d.datdba = u.oid WHERE rolname = $1;", user.Name)
		if err != nil {
			return NewError(err.Error()).Status(ErrFailedOperation)
		}

		var datnames []string
		for rows.Next() {
			var datname string
			err := rows.Scan(&datname)
			if err != nil {
				return NewError(err.Error()).Status(ErrFailedOperation)
			}
			datnames = append(datnames, datname)
		}
		rows.Close()

		for _, datname := range datnames {
			_, err := m.db.Exec("DROP DATABASE " + datname)
			if err != nil {
				return NewError("Failure while dropping database").Append(err.Error()).Status(ErrFailedOperation)
			}
		}

		dropUserQuery := m.parseDropUserQuery(user)
		_, err = m.db.Exec(dropUserQuery.Query)
		if err != nil {
			return NewError(err.Error()).Status(ErrFailedOperation)
		}
	}
	return nil
}
func (m *Commiter) dropDatabase(database *DatabaseRegistor) *Error {
	_, err := m.db.Exec(fmt.Sprintf("DROP DATABASE %s;", database.Name))
	if err != nil {
		return NewError(err.Error()).Status(ErrFailedOperation)
	}
	return nil
}
func (m *Commiter) parseCreateDatabaseQuery(database *DatabaseRegistor) *Query {
	q := Query{}
	q.Query = fmt.Sprintf("CREATE DATABASE %s WITH OWNER = %s;", database.Name, database.Owner.Name)
	return &q
}
func (m *Commiter) parseCreateUserQuery(user *User) *Query {
	q := Query{}
	q.Query = fmt.Sprintf("CREATE USER %s\n\tWITH LOGIN\n\tPASSWORD '%s';", user.Name, user.Password())
	return &q
}
func (m *Commiter) parseDropUserQuery(user *User) *Query {
	q := Query{}
	q.Query = fmt.Sprintf("DROP USER %s;", user.Name)
	return &q
}
