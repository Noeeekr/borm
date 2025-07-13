package borm

import (
	"fmt"
	"strings"
)

type Type int

const (
	INSERT Type = iota
	UPDATE
	SELECT
	DELETE
)

type Query struct {
	typ                 Type
	requiredValueLength int
	values              []any

	hasWhere bool
	hasSet   bool

	Query       string
	Information *Table
	Error       *Error
}

func (q *Query) SetError(e *Error) *Query {
	q.Error = e
	return q
}

func Update(table any) *Query {
	q := newQueryOnTable(table)
	q.typ = UPDATE
	return q
}
func Select(table any, fieldsName ...string) *Query {
	q := newQueryOnTable(table)
	if q.Error != nil {
		return q
	}
	fields, err := q.findFieldsByName(fieldsName...)
	if err != nil {
		q.Error = err
		return q
	}
	q.typ = SELECT
	q.Query = fmt.Sprintf("SELECT %s FROM %s ", strings.Join(fields, ", "), q.Information.Name)
	return q
}

func Insert(table any, fieldsName ...string) *Query {
	q := newQueryOnTable(table)
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
	q.Query = fmt.Sprintf("INSERT INTO %s (%s) ", q.Information.Name, strings.Join(fields, ", "))
	return q
}
func (q *Query) Values(values ...any) *Query {
	if q.Error != nil {
		return q
	}
	if q.typ == SELECT || q.typ == DELETE {
		q.Error = NewError().Status(ErrInvalidMethodChain).Description("Must be INSERT or UPDATE")
		return q
	}
	var valueAmount = len(values)
	if valueAmount == 0 {
		q.Error = NewError().Description("Cannot use empty values").Status(ErrEmpty)
		return q
	}
	if valueAmount%q.requiredValueLength != 0 {
		q.Error = NewError().
			Description("Invalid value amount").
			Append(fmt.Sprintf("Wanted: multiple of %d. Recieved: %d", q.requiredValueLength, valueAmount)).
			Status(ErrSyntax)
		return q
	}

	// Create a postgres value placeholder
	var fieldIndex int = 1
	fields := make([]string, valueAmount/q.requiredValueLength)
	for i := range fields {
		fieldValues := make([]string, q.requiredValueLength)
		for j := range fieldValues {
			fieldValues[j] = fmt.Sprintf("$%d", fieldIndex)
			fieldIndex++
		}
		fields[i] = fmt.Sprintf("(%s)", strings.Join(fieldValues, ", "))
	}

	q.values = values
	q.Query += fmt.Sprintf("VALUES %s ", strings.Join(fields, ","))
	return q
}
func (q *Query) Set(field string, value any) *Query {
	if q.typ != UPDATE {
		q.Error = NewError().Status(ErrInvalidMethodChain).Description("Must be INSERT or UPDATE")
		return q
	}

	if q.hasSet {
		q.Query += "SET "
		q.hasSet = true
	} else {
		q.Query += ", "
	}

	switch value.(type) {
	case string:
		q.Query += fmt.Sprintf("%s = '%s'", field, value)
	default:
		q.Query += fmt.Sprintf("%s = %d", field, value)
	}

	return q
}

type WhereCondition struct {
	Field string
	Value any
}

func (q *Query) Where(fieldName string, fieldValue any) *Query {
	if q.Error != nil {
		return q
	}
	if q.typ == INSERT {
		q.Error = NewError().Status(ErrInvalidMethodChain).Description("Must be INSERT | UPDATE | DELETE")
		return q
	}
	if _, err := q.findFieldsByName(fieldName); err != nil {
		q.Error = err
		return q
	}

	if q.hasWhere {
		q.Query += "AND "
	} else {
		q.Query += "WHERE "
		q.hasWhere = true
	}

	switch fieldValue.(type) {
	case string:
		q.Query += fmt.Sprintf("%s = '%s' ", fieldName, fieldValue)
	default:
		q.Query += fmt.Sprintf("%s = %d ", fieldName, fieldValue)
	}

	return q
}

func Delete(table any) *Query {
	q := newQueryOnTable(table)
	if q.Error != nil {
		return q
	}
	q.typ = DELETE
	q.Query += fmt.Sprintf("DELETE FROM %s ", q.Information.Name)
	return q
}

func newQueryOnTable(t any) *Query {
	var q Query
	tableInformation, err := tables.Table(t)
	if err != nil {
		return q.SetError(err)
	}
	q.Information = tableInformation
	return &q
}

func (q *Query) findFieldsByName(fieldsName ...string) ([]string, *Error) {
	var fields []string
	for _, fieldName := range fieldsName {
		_, exists := q.Information.Fields[fieldName]
		if !exists {
			return nil, NewError().Status(ErrNotFound).Description(fieldName + " does not exist in " + q.Information.Name)
		}
		fields = append(fields, fieldName)
	}
	return fields, nil
}
