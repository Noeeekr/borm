package borm

import (
	"fmt"

	"github.com/Noeeekr/borm/configuration"
)

// Environment migrates the environment if migrations is enabled and then attempts to connect to the database. If migrations is not enabled it jumps to the connection.
func (m *Commiter) MigrateUsers(users ...*User) error {
	migrations := configuration.Settings().Migrations()
	if !migrations.Enabled {
		return ErrorDescription(ErrConfiguration, "Must enable migrations in settings first")
	}

	created, err := m.migrateUsers(users...)
	if err != nil {
		if migrations.Undo {
			return m.dropDatabaseUsers(created...)
		}
		return err
	}

	return nil
}
func (m *Commiter) MigrateDatabase(registor *DatabaseRegistry) (*Commiter, error) {
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

func (m *Commiter) migrateUsers(users ...*User) ([]*User, error) {
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
func (m *Commiter) migrateDatabaseUser(user *User) error {
	rows, err := m.db.Query("SELECT rolname FROM pg_catalog.pg_roles WHERE rolname = $1;", user.Name)
	if err != nil {
		return ErrorDescription(ErrSyntax, err.Error())
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
	_, err = m.db.Exec(createUserQuery.build())
	if err != nil {
		return ErrorDescription(ErrFailedOperation, err.Error())
	}
	return nil
}
func (m *Commiter) migrateDatabase(database *DatabaseRegistry) error {
	rows, err := m.db.Query("SELECT datname FROM pg_catalog.pg_database WHERE datname = $1;", database.Name)
	if err != nil {
		return ErrorDescription(ErrSyntax, err.Error())
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

	_, err = m.db.Exec(m.parseCreateDatabaseQuery(database).build())
	if err != nil {
		return ErrorDescription(ErrFailedOperation, err.Error())
	}
	return nil
}
func (m *Commiter) dropDatabaseUsers(users ...*User) error {
	for _, user := range users {
		rows, err := m.db.Query("SELECT datname FROM pg_catalog.pg_database d INNER JOIN pg_catalog.pg_roles u ON d.datdba = u.oid WHERE rolname = $1;", user.Name)
		if err != nil {
			return ErrorDescription(ErrFailedOperation, err.Error())
		}

		var datnames []string
		for rows.Next() {
			var datname string
			err := rows.Scan(&datname)
			if err != nil {
				return ErrorDescription(ErrFailedOperation, err.Error())
			}
			datnames = append(datnames, datname)
		}
		rows.Close()

		for _, datname := range datnames {
			_, err := m.db.Exec("DROP DATABASE " + datname)
			if err != nil {
				return ErrorDescription(ErrFailedOperation, err.Error())
			}
		}

		dropUserQuery := m.parseDropUserQuery(user)
		_, err = m.db.Exec(dropUserQuery.build())
		if err != nil {
			return ErrorDescription(ErrFailedOperation, err.Error())
		}
	}
	return nil
}
func (m *Commiter) dropDatabase(database *DatabaseRegistry) error {
	_, err := m.db.Exec(fmt.Sprintf("DROP DATABASE %s;", database.Name))
	if err != nil {
		return ErrorDescription(ErrFailedOperation, err.Error())
	}
	return nil
}
func (m *Commiter) parseCreateDatabaseQuery(database *DatabaseRegistry) *Query {
	return newUnsafeQuery(CREATE, fmt.Sprintf("CREATE DATABASE %s WITH OWNER = %s;", database.Name, database.Owner.Name))
}
func (m *Commiter) parseCreateUserQuery(user *User) *Query {
	return newUnsafeQuery(CREATE, fmt.Sprintf("CREATE USER %s\n\tWITH LOGIN\n\tPASSWORD '%s';", user.Name, user.Password()))
}
func (m *Commiter) parseDropUserQuery(user *User) *Query {
	return newUnsafeQuery(DROP, fmt.Sprintf("DROP USER %s;", user.Name))
}
