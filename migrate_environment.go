package borm

import (
	"fmt"

	"github.com/Noeeekr/borm/configuration"
)

// Environment migrates the environment if migrations is enabled and then attempts to connect to the database. If migrations is not enabled it jumps to the connection.
func (m *Commiter) MigrateUsers(users ...*User) *Error {
	migrations := configuration.Settings().Migrations()
	if !migrations.Enabled {
		return NewError("Migration must be enabled first").Status(ErrConfiguration)
	}

	created, err := m.migrateUsers(users...)
	if err != nil {
		if migrations.Undo {
			return err.Join(m.dropDatabaseUsers(created...))
		}
		return err
	}

	return nil
}
func (m *Commiter) MigrateDatabase(registor *DatabaseRegistry) (*Commiter, *Error) {
	migrations := configuration.Settings().Migrations()
	if !migrations.Enabled {
		return Connect(registor)
	}

	err := m.migrateDatabase(registor)
	if err != nil {
		return nil, err
	}

	return Connect(registor)
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
func (m *Commiter) migrateDatabase(database *DatabaseRegistry) *Error {
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
func (m *Commiter) dropDatabase(database *DatabaseRegistry) *Error {
	_, err := m.db.Exec(fmt.Sprintf("DROP DATABASE %s;", database.Name))
	if err != nil {
		return NewError(err.Error()).Status(ErrFailedOperation)
	}
	return nil
}
func (m *Commiter) parseCreateDatabaseQuery(database *DatabaseRegistry) *Query {
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
