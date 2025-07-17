package manager

import (
	"fmt"
	"strings"

	"github.com/Noeeekr/borm/configuration"
	"github.com/Noeeekr/borm/errors"
	"github.com/Noeeekr/borm/internal/registers"
	"github.com/Noeeekr/borm/internal/transaction"
)

// returns nil if migrations are not enabled in settings
func (m *DatabaseManager) Relations() *errors.Error {
	if !configuration.Settings().Migrations().Enabled {
		return nil
	}
	manager := transaction.NewManager(m.db)
	t, err := manager.Start()
	if err != nil {
		return err
	}

	if err := m.migrateTables(t); err != nil {
		return errors.New("Unable to migrate tables").Join(err).Status(errors.ErrFailedTransaction)
	}

	return t.Commit()
}
func (m *DatabaseManager) validateTables() *errors.Error {
	for _, table := range *m.Register.TableCache {
		if table.Error != nil {
			return table.Error
		}
	}
	return nil
}
func (m *DatabaseManager) migrateTables(t *transaction.Transaction) *errors.Error {
	if err := m.validateTables(); err != nil {
		return err
	}

	for _, table := range *m.Register.TableCache {
		if err := m.migrateTable(t, table); err != nil {
			return err
		}
	}

	return nil
}
func (m *DatabaseManager) migrateTable(t *transaction.Transaction, table *registers.Table) *errors.Error {
	var exists bool
	existsQuery := registers.NewQuery("SELECT tablename FROM pg_catalog.pg_tables WHERE tablename = $1").
		Scanner(transaction.CheckExist(&exists))
	existsQuery.CurrentValues = append(existsQuery.CurrentValues, table.TableName)
	err := t.Do(existsQuery)
	if err != nil {
		return err
	}
	configuration := configuration.Settings().Migrations()
	if exists && configuration.Ignore {
		return nil
	}
	if exists && configuration.Recreate {
		if err = m.dropTable(t, table); err != nil {
			return err
		}
	}

	for _, role := range table.RequiredRoles {
		if ok := m.cache[string(role.RoleName())]; ok {
			continue
		}

		if role.RoleType() == registers.ENUM {
			enum := role.(*registers.Enum)
			err = m.migrateEnum(t, enum)
		}
		// error separated since the if above can become a switch with many role types
		if err != nil {
			return err
		}

		m.cache[string(role.RoleName())] = true
	}

	for _, subtable := range table.RequiredTables {
		if ok := m.cache[string(subtable.TableName)]; ok {
			continue
		}

		err = m.migrateTable(t, subtable)
		if err != nil {
			return err
		}

		m.cache[string(subtable.TableName)] = true
	}

	m.cache[string(table.TableName)] = true

	query := parseCreateTableQuery(table)
	return t.Do(query)
}
func (m *DatabaseManager) migrateEnum(t *transaction.Transaction, enum *registers.Enum) *errors.Error {
	var exists bool
	registers.
		NewQuery("SELECT typtype FROM pg_catalog.pg_type WHERE typtype = 'e' AND typname = $1").
		Values(enum.Name).
		Scanner(transaction.CheckExist(&exists))

	configuration := configuration.Settings().Migrations()
	if exists && configuration.Ignore {
		return nil
	}
	if exists && configuration.Recreate {
		err := m.dropEnum(t, enum)
		if err != nil {
			return err
		}
	}
	return t.Do(parseCreateEnumQuery(enum))
}
func (m *DatabaseManager) dropEnum(t *transaction.Transaction, enum *registers.Enum) *errors.Error {
	query := registers.NewQuery("DROP TYPE $1;")
	query.CurrentValues = append(query.CurrentValues, enum.Name)
	return t.Do(query)
}
func (m *DatabaseManager) dropTable(t *transaction.Transaction, table *registers.Table) *errors.Error {
	query := registers.NewQuery("DROP TABLE $1 CASCADE;")
	query.CurrentValues = append(query.CurrentValues, table.Name)

	t.Do(query)

	return nil
}
func parseCreateTableQuery(table *registers.Table) *registers.Query {
	var fields []string
	for _, field := range table.Fields {
		query := fmt.Sprintf("\n\t%s %s", field.Name, field.Type)
		if field.Constraints != "" {
			query += fmt.Sprintf(" %s", field.Constraints)
		}
		if field.ForeignKey != "" {
			query += fmt.Sprintf(",%s", field.ForeignKey)
		}
		fields = append(fields, query)
	}

	return registers.NewQuery(fmt.Sprintf("CREATE TABLE %s (%s\n);", table.TableName, strings.Join(fields, ",")))
}
func parseCreateEnumQuery(enum *registers.Enum) *registers.Query {
	values := enum.Values()
	for i, value := range values {
		values[i] = fmt.Sprintf("'%s'", strings.ToLower(value))
	}

	queryString := fmt.Sprintf("CREATE TYPE %s AS ENUM (%s);", strings.ToLower(string(enum.Name)), strings.Join(values, ", "))
	query := registers.NewQuery(queryString)
	return query
}
