package borm

import (
	"fmt"
	"maps"
	"reflect"
	"strings"
	"time"
)

// Used internally to split borm tag fields
const TAG_L_TRIM_QNT int = 1
const TAG_R_TRIM_QNT int = 1
const TAG_FIELDS_SEPARATOR string = ") ("

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
	Fields    map[TableColumnName]*TableColumns
	Error     *Error

	RequiredTypes  []TypMethods
	RequiredTables []*TableRegistry

	databaseCache *TablesCache
}

type TableColumnName string
type TableColumns struct {
	Name        TableColumnName
	Type        string
	Constraints string
	ForeignKey  string
}

func NewTableRegistry(name string) *TableRegistry {
	return &TableRegistry{
		TableName: TableName(strings.ToLower(name)),
		Fields:    make(map[TableColumnName]*TableColumns),
	}
}

// Registers a table for migration and queries. v must be a struct type.
func (m *TablesCache) RegisterTable(v any) *TableRegistry {
	// Check if the value is a struct
	typ := reflect.TypeOf(v)
	if typ.Kind() != reflect.TypeFor[struct{}]().Kind() {
		var t TableRegistry
		t.Error = NewError(typ.Name() + " must be of kind struct").Status(ErrInvalidType)
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
			t.Error = NewError("TableRegistry is not registered. Unable to use it as a dependency.").
				Status(ErrNotFound)
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
	q.tableAliases[""] = m
	q.fields = append(q.fields, fieldsName...)

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
	q.fields = append(q.fields, fieldsName...)

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

// Breaks the borm tag of a field and parses its values into query parts
type ColumnTagReader struct {
	// Used by Override() to set a TableColumns to recieve the values if present
	mockValues *TableColumns
}

type Tag struct {
	mockValues *TableColumns
	values     map[TableColumnName][]string
}

func newColumnTagReader() *ColumnTagReader {
	return &ColumnTagReader{
		mockValues: nil,
	}
}

func NewTag() *Tag {
	return &Tag{
		mockValues: &TableColumns{},
		values:     map[TableColumnName][]string{},
	}
}
func (m *ColumnTagReader) WriteTo(f *TableColumns) *ColumnTagReader {
	m.mockValues = f
	return m
}
func (m *ColumnTagReader) NewTagValues(tag string) *Tag {
	var tagFields []string
	if tag != "" {
		tagFields = strings.Split(tag[TAG_L_TRIM_QNT:len(tag)-TAG_R_TRIM_QNT], TAG_FIELDS_SEPARATOR)
	}

	// Trim tag whitespaces
	for index := range tagFields {
		tagFields[index] = strings.TrimSpace(tagFields[index])
	}

	// Separate tag fields into keys and values
	fieldTag := NewTag()
	for _, Tablefield := range tagFields {
		TablefieldValues := strings.Split(Tablefield, ",")
		if len(TablefieldValues) < 2 {
			continue
		}
		// Trim field whitespaces
		for i, value := range TablefieldValues {
			TablefieldValues[i] = strings.ToLower(strings.TrimSpace(value))
		}
		fieldName := TableColumnName(strings.ToUpper(TablefieldValues[0]))
		fieldValues := TablefieldValues[1:]
		fieldTag.values[fieldName] = fieldValues
	}

	return fieldTag
}
func (m *ColumnTagReader) ReadFrom(f reflect.StructField) *TableColumns {
	tagValues := m.NewTagValues(f.Tag.Get("borm"))
	tagValues.FillWith(m.mockValues)

	field := &TableColumns{}
	field.Name = tagValues.parseName()
	field.Type = tagValues.parseType()
	field.Constraints = tagValues.parseConstraints()
	field.ForeignKey = tagValues.parseForeignKey(field.Name)

	return field
}

func (t *Tag) FillWith(tf *TableColumns) {
	t.mockValues = tf
}

func (t *Tag) parseName() TableColumnName {
	if values := t.values["NAME"]; len(values) > 0 {
		return TableColumnName(values[0])
	}
	return t.mockValues.Name
}
func (t *Tag) parseType() string {
	if values := t.values["TYPE"]; len(values) > 0 {
		return values[0]
	}
	return t.mockValues.Type
}
func (t *Tag) parseConstraints() string {
	values := t.values["CONSTRAINTS"]
	return strings.Join(values, " ")
}
func (t *Tag) parseForeignKey(f TableColumnName) string {
	values := t.values["FOREIGN KEY"]
	if len(values) < 2 {
		return ""
	}

	var foreignKey string = fmt.Sprintf("\n\tFOREIGN KEY (%s)\n\tREFERENCES %s (%s)", f, values[0], values[1])

	values = t.values["UPDATE"]
	if len(values) > 0 {
		foreignKey += fmt.Sprintf("\n\tON UPDATE %s", strings.ToUpper(values[0]))
	}

	values = t.values["DELETE"]
	if len(values) > 0 {
		foreignKey += fmt.Sprintf("\n\tON DELETE %s", strings.ToUpper(values[0]))
	}

	return foreignKey
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
	columns := map[TableColumnName]*TableColumns{}

	columnTagReader := newColumnTagReader()
	for i := range typ.NumField() {
		field := typ.Field(i)
		typ := field.Type
		if typ.Kind() == reflect.Pointer {
			typ = typ.Elem()
		}
		// Copy the embedded struct fields
		if field.Anonymous && typ.Kind() == reflect.Struct {
			maps.Copy(columns, parseFields(typ))
			continue
		}
		columnName := strings.ToLower(field.Name)
		columnType := parseFieldType(typ.Name())
		column := columnTagReader.
			WriteTo(newTableColumns(columnName, columnType)).
			ReadFrom(field)

		columns[column.Name] = column
	}
	return columns
}
func newTableColumns(name string, typ string) *TableColumns {
	return &TableColumns{
		Name: TableColumnName(name),
		Type: typ,
	}
}
