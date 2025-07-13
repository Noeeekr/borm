package query

import (
	"fmt"
	"strings"

	"github.com/Noeeekr/borm/common"
	"github.com/Noeeekr/borm/table"
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
	hasWhere            bool

	Query       string
	Information *table.TableInformation
	Error       *common.Error
}

func (q *Query) SetError(e *common.Error) *Query {
	q.Error = e
	return q
}

func Select(table any, fieldsName ...string) *Query {
	q := newWithTableInformation(table)
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
	q := newWithTableInformation(table)
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
	if q.typ == SELECT || q.typ == DELETE {
		q.Error = common.NewError().Status(common.ErrInvalidMethodChain).Description("Must be INSERT or UPDATE")
		return q
	}
	var valueAmount = len(values)
	if valueAmount == 0 {
		q.Error = common.NewError().Description("Cannot use empty values").Status(common.ErrEmpty)
		return q
	}
	if valueAmount%q.requiredValueLength != 0 {
		q.Error = common.NewError().
			Description("Invalid value amount").
			Append(fmt.Sprintf("Wanted: multiple of %d. Recieved: %d", q.requiredValueLength, valueAmount)).
			Status(common.ErrSyntax)
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

type WhereCondition struct {
	Field string
	Value any
}

func (q *Query) Where(fieldName string, fieldValue any) *Query {
	if q.typ == INSERT {
		q.Error = common.NewError().Status(common.ErrInvalidMethodChain).Description("Must be INSERT | UPDATE | DELETE")
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
	q := newWithTableInformation(table)
	if q.Error != nil {
		return q
	}
	q.typ = DELETE
	q.Query += fmt.Sprintf("DELETE FROM %s ", q.Information.Name)
	return q
}

func (q *Query) findFieldsByName(fieldsName ...string) ([]string, *common.Error) {
	var fields []string
	for _, fieldName := range fieldsName {
		_, exists := q.Information.Fields[fieldName]
		if !exists {
			return nil, common.NewError().Status(common.ErrNotFound).Description(fieldName + " does not exist in " + q.Information.Name)
		}
		fields = append(fields, fieldName)
	}
	return fields, nil
}

func newWithTableInformation(t any) *Query {
	var q Query
	tableInformation, err := table.GetInformation(t)
	if err != nil {
		return q.SetError(err)
	}
	q.Information = tableInformation
	return &q
}
