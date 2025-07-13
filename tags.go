package borm

import (
	"fmt"
	"strings"
)

// Breaks the borm tag of a field and parses its values into query parts
type FieldTagParser struct {
	// Used by Override() to set a TableField to recieve the values if present
	mockValues *TableField
}

type Tag struct {
	mockValues *TableField
	values     map[string][]string
}

// Used internally to split borm tag fields
const TAG_L_TRIM_QNT int = 1
const TAG_R_TRIM_QNT int = 1
const TAG_FIELDS_SEPARATOR string = ") ("

func newFieldTagParser() *FieldTagParser {
	return &FieldTagParser{
		mockValues: nil,
	}
}

func newFieldTag() *Tag {
	return &Tag{
		mockValues: &TableField{},
		values:     map[string][]string{},
	}
}
func (m *FieldTagParser) Override(f *TableField) *FieldTagParser {
	m.mockValues = f
	return m
}
func (m *FieldTagParser) NewFieldTagParser(tag string) *Tag {
	var tagFields []string
	if tag != "" {
		tagFields = strings.Split(tag[TAG_L_TRIM_QNT:len(tag)-TAG_R_TRIM_QNT], TAG_FIELDS_SEPARATOR)
	}

	// Trim tag whitespaces
	for index := range tagFields {
		tagFields[index] = strings.TrimSpace(tagFields[index])
	}

	// Separate tag fields into keys and values
	fieldTag := newFieldTag()
	for _, Tablefield := range tagFields {
		TablefieldValues := strings.Split(Tablefield, ",")
		if len(TablefieldValues) < 2 {
			continue
		}
		// Trim field whitespaces
		for i, value := range TablefieldValues {
			TablefieldValues[i] = strings.ToLower(strings.TrimSpace(value))
		}
		fieldName := strings.ToUpper(TablefieldValues[0])
		fieldValues := TablefieldValues[1:]
		fieldTag.values[fieldName] = fieldValues
	}

	return fieldTag
}
func (m *FieldTagParser) ParseRaw(tag string) *TableField {
	tagValues := m.NewFieldTagParser(tag)
	tagValues.FillEmptyWithFieldValues(m.mockValues)

	field := &TableField{}
	field.Name = tagValues.ParseName()
	field.Type = tagValues.ParseType()
	field.Constraints = tagValues.ParseConstraints()
	field.ForeignKey = tagValues.ParseForeignKey(field.Name)

	return field
}

func (t *Tag) FillEmptyWithFieldValues(tf *TableField) {
	t.mockValues = tf
}

// Uses mock if value is empty
func (t *Tag) ParseName() string {
	if values := t.values["NAME"]; len(values) > 0 {
		return values[0]
	}
	return t.mockValues.Name
}
func (t *Tag) ParseType() string {
	if values := t.values["TYPE"]; len(values) > 0 {
		return values[0]
	}
	return t.mockValues.Type
}
func (t *Tag) ParseConstraints() string {
	values := t.values["CONSTRAINTS"]
	return strings.Join(values, " ")
}
func (t *Tag) ParseForeignKey(fieldName string) string {
	values := t.values["FOREIGN KEY"]
	if len(values) < 2 {
		return ""
	}

	var foreignKey string = fmt.Sprintf("\n\tFOREIGN KEY (%s)\n\tREFERENCES %s (%s)", fieldName, values[0], values[1])

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
