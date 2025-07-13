package table

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/Noeeekr/borm/common"
)

var tableCache = map[string]*TableInformation{}

type TableInformation struct {
	Name   string
	Fields map[string]*Field
}

type Field struct {
	Name        string
	Type        string
	Constraints string
	ForeignKey  string
}

type MappedTags map[string][]string

func (m *MappedTags) GetFirst(v string) (string, bool) {
	if values, ok := (*m)[v]; ok {
		return values[0], true
	}
	return "", false
}
func (m *MappedTags) GetN(v string, n int) ([]string, bool) {
	if values, ok := (*m)[v]; ok && len(values) >= n {
		return values[:n], true
	}
	return nil, false
}
func (m *MappedTags) GetAll(v string) []string {
	return (*m)[v]
}

// Retuns tableName and tableField information
func GetInformation(v any) (*TableInformation, *common.Error) {
	typ := reflect.TypeOf(v)
	if res := common.IsStruct(typ); res != nil {
		return nil, res
	}
	tableName := strings.ToLower(typ.Name())
	if tableInformation, ok := tableCache[tableName]; ok {
		return tableInformation, nil
	}

	information := &TableInformation{
		Name:   tableName,
		Fields: map[string]*Field{},
	}
	for i := range typ.NumField() {
		fieldType := typ.Field(i)
		field := parseFieldTags(&fieldType)
		information.Fields[field.Name] = field
	}

	tableCache[tableName] = information

	return information, nil
}
func parseTagStringIntoMappedTags(rawString string) MappedTags {
	// Separate tags by the ) parenthesis
	var fields []string
	if rawString != "" {
		fields = strings.Split(rawString[1:len(rawString)-1], ") (")
	}

	// Trim whitespaces and left parenthesis
	for index := range fields {
		fields[index] = strings.TrimSpace(fields[index])
	}

	// Separate the keys and values
	mappedTags := MappedTags{}
	for _, field := range fields {
		fieldValues := strings.Split(field, ",")
		if len(fieldValues) < 2 {
			continue
		}
		// Trim whitespaces
		for i, value := range fieldValues {
			fieldValues[i] = strings.ToLower(strings.TrimSpace(value))
		}
		mappedTags[strings.ToUpper(fieldValues[0])] = fieldValues[1:]
	}

	return mappedTags
}
func parseFieldTags(f *reflect.StructField) *Field {
	rawTagString := f.Tag.Get("borm")
	mappedTags := parseTagStringIntoMappedTags(rawTagString)
	// Optinal value.. Reflect the field name to lowercase if not present

	// Optinal value.. Reflect the field type if not present
	typ, _ := mappedTags.GetFirst("TYPE")
	if typ == "" {
		typ = getFieldType(f)
	}
	constraints := strings.Join(mappedTags.GetAll("CONSTRAINTS"), "")

	name, _ := mappedTags.GetFirst("NAME")
	if name == "" {
		name = f.Name
	}

	field := &Field{}
	field.Name = strings.ToLower(name)
	field.Type = typ
	field.Constraints = constraints
	if targets, ok := mappedTags.GetN("FOREIGN KEY", 2); ok {
		deleteAction, ok := mappedTags.GetFirst("DELETE")
		if ok {
			deleteAction = strings.ToUpper(deleteAction)
		}

		updateAction, ok := mappedTags.GetFirst("UPDATE")
		if ok {
			updateAction = strings.ToUpper(updateAction)
		}
		field.ForeignKey = parseFieldForeignKey(field, targets[0], targets[1], updateAction, deleteAction)
	}
	return field
}
func parseFieldForeignKey(field *Field, targetTable, targetField, updateAction, deleteAction string) (foreignKey string) {
	foreignKey += fmt.Sprintf("\n\tFOREIGN KEY (%s) \n\tREFERENCES %s (%s)", field.Name, targetTable, targetField)

	// Set actions if present
	if updateAction != "" {
		foreignKey += fmt.Sprintf("\n\tON UPDATE %s", updateAction)
	}
	if deleteAction != "" {
		foreignKey += fmt.Sprintf("\n\tON DELETE %s", deleteAction)
	}
	return foreignKey
}
func getFieldType(f *reflect.StructField) string {
	if typ, ok := f.Tag.Lookup("type"); ok {
		return typ
	}

	switch f.Type.Name() {
	case reflect.TypeFor[string]().Name():
		return "VARCHAR(256)"
	case reflect.TypeFor[int]().Name():
		return "INTEGER"
	case reflect.TypeFor[time.Time]().Name():
		return "TIMESTAMPTZ"
	default:
		return strings.ToLower(f.Type.Name())
	}
}
