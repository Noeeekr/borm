package borm

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

// Breaks the borm tag of a field and parses its values into query parts
type TagReader struct {
	writeTarget *TableFieldValues
}

type Tag struct {
	DefaultValues *TableFieldValues
	values        map[TableFieldName][]string
}

func newTagReader() *TagReader {
	return &TagReader{
		writeTarget: nil,
	}
}

func newTag() *Tag {
	return &Tag{
		DefaultValues: &TableFieldValues{},
		values:        map[TableFieldName][]string{},
	}
}
func (m *TagReader) Override(f *TableFieldValues) *TagReader {
	m.writeTarget = f
	return m
}
func (m *TagReader) ParseStringTag(tag string) *Tag {
	var (
		LEFT_TRIM        = 1
		RIGHT_TRIM       = len(tag) - 1
		FIELDS_SEPARATOR = ") ("
	)

	var fields []string
	if tag != "" {
		fields = strings.Split(tag[LEFT_TRIM:RIGHT_TRIM], FIELDS_SEPARATOR)
	}

	// Trim tag whitespaces
	for index := range fields {
		fields[index] = strings.TrimSpace(fields[index])
	}

	fieldTag := newTag()
	for _, Tablefield := range fields {
		// Break tag fields into keys and values
		TablefieldValues := strings.Split(Tablefield, ",")

		// For booleans
		fieldName := TableFieldName(strings.ToUpper(TablefieldValues[0]))
		if len(TablefieldValues) == 1 {
			fieldTag.values[fieldName] = []string{"-"}
			continue
		}

		// Trim field whitespaces
		for i, value := range TablefieldValues {
			TablefieldValues[i] = strings.ToLower(strings.TrimSpace(value))
		}
		fieldValues := TablefieldValues[1:]
		fieldTag.values[fieldName] = fieldValues
	}

	return fieldTag
}
func (m *TagReader) Read(f reflect.StructField) *TableFieldValues {
	tag := m.ParseStringTag(f.Tag.Get("borm"))
	tag.UseDefaulValues(m.writeTarget)

	field := newTableFieldValues(tag.GetName(), tag.GetType())
	field.Constraints = tag.GetConstraints()
	field.ForeignKey = tag.GetForeignKey(field.Name)
	field.Ignore = tag.GetIgnore()

	return field
}

func (t *Tag) UseDefaulValues(tf *TableFieldValues) {
	t.DefaultValues = tf
}
func (t *Tag) GetIgnore() bool {
	if values := t.values["IGNORE"]; len(values) > 0 {
		return true
	}
	return false
}
func (t *Tag) GetName() TableFieldName {
	if values := t.values["NAME"]; len(values) > 0 {
		return TableFieldName(values[0])
	}
	return t.DefaultValues.Name
}
func (t *Tag) GetType() string {
	if values := t.values["TYPE"]; len(values) > 0 {
		return values[0]
	}
	return t.DefaultValues.Type
}
func (t *Tag) GetConstraints() string {
	values := t.values["CONSTRAINTS"]
	return strings.Join(values, " ")
}
func (t *Tag) GetForeignKey(f TableFieldName) string {
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
