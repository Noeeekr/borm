package borm

import (
	"reflect"
	"strings"
	"time"
)

var tables = newTables()

type Tables struct {
	cache map[string]*Table
}

type Table struct {
	Name   string
	Fields map[string]*TableField
}

type TableField struct {
	Name        string
	Type        string
	Constraints string
	ForeignKey  string
}

func newTables() *Tables {
	return &Tables{
		cache: map[string]*Table{},
	}
}

// Retuns tableName and tableTableField information
func (m *Tables) Table(v any) (*Table, *Error) {
	typ := reflect.TypeOf(v)
	if res := isStruct(typ); res != nil {
		return nil, res
	}
	tableName := strings.ToLower(typ.Name())
	if Table, ok := m.cache[tableName]; ok {
		return Table, nil
	}

	information := &Table{
		Name:   tableName,
		Fields: map[string]*TableField{},
	}

	tagParser := newFieldTagParser()
	for i := range typ.NumField() {
		field := typ.Field(i)
		fieldInformation := &TableField{}
		fieldInformation.Name = strings.ToLower(field.Name)
		fieldInformation.Type = parseFieldType(field.Type.Name())
		tableField := tagParser.Override(fieldInformation).ParseRaw(string(field.Tag))
		information.Fields[tableField.Name] = tableField
	}

	m.cache[tableName] = information

	return information, nil
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
		return typname
	}
}
