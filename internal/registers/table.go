package registers

import (
	"fmt"
	"maps"
	"reflect"
	"strings"
	"time"

	"github.com/Noeeekr/borm/common"
	"github.com/Noeeekr/borm/errors"
)

const (
	SELECT TablePrivilege = iota
	UPDATE
	DELETE
	INSERT
	DROP

	ALL
)

type TableName string
type TableCache map[TableName]*Table
type TablePrivilege int
type Table struct {
	TableName TableName
	Fields    map[TableColumnName]*TableColumns
	Error     *errors.Error

	RequiredRoles  []RoleMethods
	RequiredTables []*Table

	// (RoleName) can have (TablePrivileges) on columns (TableColumnName)
	privileges map[RoleName]map[TableColumnName]TablePrivilege

	cache *TableCache
}

type TableColumnName string
type TableColumns struct {
	Name        TableColumnName
	Type        string
	Constraints string
	ForeignKey  string
}

func NewTableCache() *TableCache {
	return &TableCache{}
}

// Registers a table for migration and queries
func (m *TableCache) Table(v any) *Table {
	typ := reflect.TypeOf(v)
	if err := common.IsStruct(typ); err != nil {
		t := &Table{}
		t.Error = err
		return t
	}
	tableName := TableName(strings.ToLower(typ.Name()))
	if Table, ok := (*m)[tableName]; ok {
		return Table
	}

	information := &Table{
		TableName: TableName(strings.ToLower(typ.Name())),
		Fields:    parseFields(typ),
		cache:     m,
	}
	(*m)[tableName] = information

	return information
}
func (t *Table) Name(n string) *Table {
	delete(*t.cache, t.TableName)
	t.TableName = TableName(n)
	(*t.cache)[t.TableName] = t
	return t
}
func (t *Table) NeedTables(dependencies ...*Table) *Table {
	for _, dependency := range dependencies {
		if _, ok := (*t.cache)[dependency.TableName]; !ok {
			t.Error = errors.New("Table is not registered. Unable to use it as a dependency.").
				Status(errors.ErrNotFound)
			return t
		}
	}
	t.RequiredTables = append(t.RequiredTables, dependencies...)
	return t
}
func (t *Table) NeedRoles(dependencies ...RoleMethods) *Table {
	t.RequiredRoles = append(t.RequiredRoles, dependencies...)
	return t
}

func (m *Table) Update() *Query {
	q := newQueryOnTable(m)
	q.Query += fmt.Sprintf("UPDATE %s ", m.TableName)
	q.typ = UPDATE
	return q
}
func (m *Table) Select(fieldsName ...TableColumnName) *Query {
	q := newQueryOnTable(m)
	if q.Error != nil {
		return q
	}
	fields, err := q.findFieldsByName(fieldsName...)
	if err != nil {
		q.Error = err
		return q
	}
	q.typ = SELECT
	q.Query = fmt.Sprintf("SELECT %s FROM %s ", strings.Join(fields, ", "), q.Information.TableName)
	return q
}

func (m *Table) Insert(fieldsName ...TableColumnName) *Query {
	q := newQueryOnTable(m)
	if q.Error != nil {
		return q
	}
	fields, err := q.findFieldsByName(fieldsName...)
	if err != nil {
		q.Error = err
		return q
	}
	q.typ = INSERT
	q.requiredValueLength = len(fieldsName)
	q.Query = fmt.Sprintf("INSERT INTO %s (%s) ", q.Information.TableName, strings.Join(fields, ", "))
	return q
}
func (m *Table) Delete() *Query {
	q := newQueryOnTable(m)
	if q.Error != nil {
		return q
	}
	q.typ = DELETE
	q.Query += fmt.Sprintf("DELETE FROM %s ", q.Information.TableName)
	return q
}
func parseFieldType(typname string) string {
	switch typname {
	case reflect.TypeFor[string]().Name():
		return "VARCHAR(256)"
	case reflect.TypeFor[int]().Name():
		return "INTEGER"
	case reflect.TypeFor[time.Time]().Name():
		return "TIMESTAMPTZ"
	default:
		return strings.ToLower(typname)
	}
}
func parseFields(typ reflect.Type) map[TableColumnName]*TableColumns {
	fields := map[TableColumnName]*TableColumns{}

	tagParser := NewFieldTagParser()
	for i := range typ.NumField() {
		field := typ.Field(i)
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			subfields := parseFields(field.Type)
			maps.Copy(fields, subfields)
			continue
		}
		fieldInformation := &TableColumns{}
		fieldInformation.Name = TableColumnName(strings.ToLower(field.Name))
		fieldInformation.Type = parseFieldType(field.Type.Name())
		tableField := tagParser.Override(fieldInformation).ParseRaw(string(field.Tag.Get("borm")))
		fields[tableField.Name] = tableField
	}
	return fields
}
