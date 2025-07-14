package borm

import (
	"fmt"
	"strings"

	"github.com/Noeeekr/borm/common"
	"github.com/Noeeekr/borm/internal/registers"
)

type QueryType int

type Query struct {
	typ                 registers.TablePrivilege
	requiredValueLength int
	values              []any
	placeholderIndex    int

	hasWhere bool
	hasSet   bool

	Query       string
	Information *registers.Table
	Error       *common.Error
}

func (q *Query) SetError(e *common.Error) *Query {
	q.Error = e
	return q
}

func Update(table *registers.Table) *Query {
	q := newQueryOnTable(table)
	q.Query += fmt.Sprintf("UPDATE %s ", table.Name)
	q.typ = UPDATE
	return q
}
func Select(table *registers.Table, fieldsName ...registers.TableColumnName) *Query {
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

func Insert(table *registers.Table, fieldsName ...registers.TableColumnName) *Query {
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

	fields := make([]string, valueAmount/q.requiredValueLength)
	for i := range fields {
		fieldValues := make([]string, q.requiredValueLength)
		for j := range fieldValues {
			fieldValues[j] = fmt.Sprintf("$%d", q.placeholderIndex)
			q.placeholderIndex++
		}
		fields[i] = fmt.Sprintf("(%s)", strings.Join(fieldValues, ", "))
	}

	q.values = values
	q.Query += fmt.Sprintf("VALUES %s ", strings.Join(fields, ","))
	return q
}
func (q *Query) Set(field registers.TableColumnName, value any) *Query {
	if q.Error != nil {
		return q
	}
	if q.typ != UPDATE {
		q.Error = common.NewError().Status(common.ErrInvalidMethodChain).Description("Must be INSERT or UPDATE")
		return q
	}
	if _, err := q.findFieldsByName(field); err != nil {
		q.Error = err
		return q
	}

	if q.hasSet {
		q.Query += ", "
	} else {
		q.hasSet = true
		q.Query += "SET "
	}

	q.Query += fmt.Sprintf("%s = $%d", field, q.placeholderIndex)
	q.placeholderIndex++
	q.values = append(q.values, value)
	return q
}

type WhereCondition struct {
	Field string
	Value any
}

func (q *Query) Where(fieldName registers.TableColumnName, fieldValue any) *Query {
	if q.Error != nil {
		return q
	}
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

	q.Query += fmt.Sprintf("%s = $%d ", fieldName, q.placeholderIndex)
	q.placeholderIndex++
	q.values = append(q.values, fieldValue)
	return q
}

func Delete(table *registers.Table) *Query {
	q := newQueryOnTable(table)
	if q.Error != nil {
		return q
	}
	q.typ = DELETE
	q.Query += fmt.Sprintf("DELETE FROM %s ", q.Information.Name)
	return q
}

func newQueryOnTable(t *registers.Table) *Query {
	var q Query
	if t == nil {
		q.Error = common.NewError().Description("Cannot query nil table").Status(common.ErrEmpty)
		return &q
	}
	table := (*registers.Tables)[t.Name]
	if table.Error != nil {
		return q.SetError(table.Error)
	}
	q.Information = table
	q.placeholderIndex = 1
	return &q
}

func (q *Query) findFieldsByName(fieldsName ...registers.TableColumnName) ([]string, *common.Error) {
	var fields []string
	for _, fieldName := range fieldsName {
		_, exists := q.Information.Fields[fieldName]
		if !exists {
			return nil, common.NewError().
				Status(common.ErrNotFound).
				Description(fmt.Sprintf("%s does not exist in %s", fieldName, q.Information.Name))
		}
		fields = append(fields, string(fieldName))
	}
	return fields, nil
}
