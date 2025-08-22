package borm

import (
	"fmt"
	"maps"
	"reflect"
	"strings"
)

const (
	SELECT QueryType = iota
	UPDATE
	DELETE
	INSERT

	CREATE
	DROP

	ALL
)

type TableName string
type TableRegistry struct {
	TableName TableName
	Fields    map[TableFieldName]*TableFieldValues
	Error     error

	RequiredTypes  []TypMethods
	RequiredTables []*TableRegistry

	databaseCache *TablesCache
}

type TableFieldName string
type TableFieldValues struct {
	Name        TableFieldName
	Type        string
	Constraints string
	ForeignKey  string
	Ignore      bool
}

func NewTableRegistry(name string) *TableRegistry {
	return &TableRegistry{
		TableName: TableName(strings.ToLower(name)),
		Fields:    make(map[TableFieldName]*TableFieldValues),
	}
}

// Registers a table for migration and queries. v must be a struct type.
func (m *TablesCache) RegisterTable(v any) *TableRegistry {
	// Check if the value is a struct
	typ := reflect.TypeOf(v)
	if typ.Kind() != reflect.TypeFor[struct{}]().Kind() {
		var t TableRegistry
		t.Error = ErrorDescription(ErrInvalidType, typ.Name(), "Must be of kind struct")
		return &t
	}
	// Check if the struct is already cached and returns it if so
	tableName := TableName(strings.ToLower(typ.Name()))
	if TableRegistry, ok := (*m)[tableName]; ok {
		return TableRegistry
	}

	// Creates and caches a new TableRegistry
	registry := &TableRegistry{
		TableName:     TableName(strings.ToLower(typ.Name())),
		Fields:        parseFields(typ),
		databaseCache: m,
	}
	(*m)[tableName] = registry

	return registry
}
func (t *TableRegistry) Name(n string) *TableRegistry {
	delete(*t.databaseCache, t.TableName)
	t.TableName = TableName(n)
	(*t.databaseCache)[t.TableName] = t
	return t
}
func (t *TableRegistry) NeedTables(dependencies ...*TableRegistry) *TableRegistry {
	for _, dependency := range dependencies {
		if _, ok := (*t.databaseCache)[dependency.TableName]; !ok {
			t.Error = ErrorDescription(ErrNotFound, "Unregistered table. Unable to use it as a dependency.")
			return t
		}
	}
	t.RequiredTables = append(t.RequiredTables, dependencies...)
	return t
}
func (t *TableRegistry) NeedRoles(dependencies ...TypMethods) *TableRegistry {
	t.RequiredTypes = append(t.RequiredTypes, dependencies...)
	return t
}
func (m *TableRegistry) Update() *Query {
	q := newQueryOnTable(m)
	q.tableAliases[""] = m
	q.Query += fmt.Sprintf("UPDATE %s ", m.TableName)
	q.typ = UPDATE
	return q
}
func (m *TableRegistry) Select(fieldsName ...string) *Query {
	q := newQueryOnTable(m)
	if q.Error != nil {
		return q
	}

	// Maps the register of this table as anonymous alias until it gets an alias
	q.tableAliases[""] = m
	q.requestedFields = append(q.requestedFields, fieldsName...)

	q.typ = SELECT
	q.Query = fmt.Sprintf("SELECT %s ", strings.Join(fieldsName, ", "))
	q.Query += fmt.Sprintf("FROM %s ", q.TableRegistry.TableName)
	return q
}

func (m *TableRegistry) Insert(fieldsName ...string) *Query {
	q := newQueryOnTable(m)
	if q.Error != nil {
		return q
	}
	q.tableAliases[""] = m
	q.requestedFields = append(q.requestedFields, fieldsName...)

	q.typ = INSERT
	q.requiredValueLength = len(fieldsName)
	q.Query = fmt.Sprintf("INSERT INTO %s (%s) ", q.TableRegistry.TableName, strings.Join(fieldsName, ", "))
	return q
}
func (m *TableRegistry) Delete() *Query {
	q := newQueryOnTable(m)
	if q.Error != nil {
		return q
	}
	q.tableAliases[""] = m
	q.typ = DELETE
	q.Query += fmt.Sprintf("DELETE FROM %s ", q.TableRegistry.TableName)
	return q
}

func parseFields(typ reflect.Type) map[TableFieldName]*TableFieldValues {
	fields := map[TableFieldName]*TableFieldValues{}

	tagReader := newTagReader()
	for i := range typ.NumField() {
		structField := typ.Field(i)
		typ := structField.Type
		if typ.Kind() == reflect.Pointer {
			typ = typ.Elem()
		}
		// Copy the embedded struct fields
		if structField.Anonymous && typ.Kind() == reflect.Struct {
			maps.Copy(fields, parseFields(typ))
			continue
		}
		fieldName := TableFieldName(strings.ToLower(structField.Name))
		fieldType := parseFieldType(typ.Name())
		field := tagReader.
			Override(newTableFieldValues(fieldName, fieldType)).
			Read(structField)

		fields[field.Name] = field
	}
	return fields
}
func newTableFieldValues(name TableFieldName, typ string) *TableFieldValues {
	return &TableFieldValues{
		Name: name,
		Type: typ,
	}
}
