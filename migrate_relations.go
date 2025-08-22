package borm

import (
	"fmt"
	"strings"

	"github.com/Noeeekr/borm/configuration"
)

// returns nil if migrations are not enabled in settings
func (r *Commiter) MigrateRelations() error {
	if !configuration.Settings().Migrations().Enabled {
		return ErrorDescription(ErrConfiguration, "Must enable migrations first")
	}
	manager := newTransactionFactory(r.db)
	t, err := manager.StartTx()
	if err != nil {
		return err
	}

	if err := r.migrateTables(t); err != nil {
		return ErrorDescription(ErrFailedTransaction, "", err.Error())
	}

	return t.Commit()
}
func (r *Commiter) DropRelations() error {
	if !configuration.Settings().Migrations().Enabled {
		return ErrorDescription(ErrConfiguration, "Must enable migrations first")
	}

	manager := newTransactionFactory(r.db)
	t, err := manager.StartTx()
	if err != nil {
		return err
	}

	tables := []*TableRegistry{}
	for _, table := range *r.DatabaseRegistry.TablesCache {
		tables = append(tables, table)
	}
	if err := r.dropTables(t, tables...); err != nil {
		return err
	}

	return t.Commit()
}
func (r *Commiter) validateTables() error {
	for _, table := range *r.DatabaseRegistry.TablesCache {
		if table.Error != nil {
			return table.Error
		}
	}
	return nil
}
func (r *Commiter) migrateTables(t *Transaction) error {
	if err := r.validateTables(); err != nil {
		return err
	}

	for _, table := range *r.TablesCache {
		if subErr := r.migrateTable(t, table); subErr != nil {
			err := ErrorDescription(ErrSyntax, fmt.Sprintf("Unable to migrate table %s", table.TableName))
			return ErrorJoin(err, subErr)
		}
	}

	return nil
}
func (r *Commiter) migrateTable(t *Transaction, table *TableRegistry) error {
	var exists bool
	existsQuery := NewUnsafeQuery(SELECT, "SELECT tablename FROM pg_catalog.pg_tables WHERE tablename = $1").
		Scanner(ScannerFindOne(&exists))
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
		if err = r.dropTables(t, table); err != nil {
			return err
		}
	}

	for _, typ := range table.RequiredTypes {
		if ok := r.RegistorCache[string(typ.GetName())]; ok {
			continue
		}

		if typ.GetType() == ENUM {
			enum := typ.(*Enum)
			err = r.migrateEnum(t, enum)
		}
		// error separated since the if above can become a switch with many role types
		if err != nil {
			return err
		}

		r.RegistorCache[string(typ.GetName())] = true
	}

	for _, subtable := range table.RequiredTables {
		if ok := r.RegistorCache[string(subtable.TableName)]; ok {
			continue
		}

		err = r.migrateTable(t, subtable)
		if err != nil {
			return err
		}

		r.RegistorCache[string(subtable.TableName)] = true
	}

	r.RegistorCache[string(table.TableName)] = true

	query := parseCreateTableQuery(table)
	return t.Do(query)
}
func (r *Commiter) migrateEnum(t *Transaction, enum *Enum) error {
	var exists bool
	query := NewUnsafeQuery(SELECT, fmt.Sprintf("SELECT typtype FROM pg_catalog.pg_type WHERE typtype = 'e' AND typname = '%s'", enum.Name)).
		Scanner(ScannerFindOne(&exists))

	if err := t.Do(query); err != nil {
		return err
	}

	configuration := configuration.Settings().Migrations()
	if exists && configuration.Ignore {
		return nil
	}
	if exists && configuration.Recreate {
		err := r.dropEnum(t, enum)
		if err != nil {
			return err
		}
	}
	return t.Do(parseCreateEnumQuery(enum))
}
func (r *Commiter) dropEnum(t *Transaction, enum *Enum) error {
	query := NewUnsafeQuery(DROP, fmt.Sprintf("DROP TYPE %s CASCADE", enum.Name))
	return t.Do(query)
}
func (r *Commiter) dropTables(t *Transaction, tables ...*TableRegistry) error {
	for _, table := range tables {
		query := NewUnsafeQuery(DROP, fmt.Sprintf("DROP TABLE %s CASCADE", table.TableName))
		if err := t.Do(query); err != nil {
			return err
		}
	}
	return nil
}
func parseCreateTableQuery(table *TableRegistry) *Query {
	var fieldStatements []string
	for _, field := range table.Fields {
		if field.Ignore {
			continue
		}
		statement := fmt.Sprintf("\n\t%s %s", field.Name, field.Type)
		if field.Constraints != "" {
			statement += fmt.Sprintf(" %s", field.Constraints)
		}
		if field.ForeignKey != "" {
			statement += fmt.Sprintf(",%s", field.ForeignKey)
		}
		fieldStatements = append(fieldStatements, statement)
	}

	return NewUnsafeQuery(CREATE, fmt.Sprintf("CREATE TABLE %s (%s\n);", table.TableName, strings.Join(fieldStatements, ",")))
}
func parseCreateEnumQuery(enum *Enum) *Query {
	values := enum.GetValues()
	for i, value := range values {
		values[i] = fmt.Sprintf("'%s'", value)
	}

	queryString := fmt.Sprintf("CREATE TYPE %s AS ENUM (%s);", strings.ToLower(string(enum.Name)), strings.Join(values, ", "))
	query := NewUnsafeQuery(CREATE, queryString)
	return query
}
