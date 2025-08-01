package borm

import (
	"database/sql"
	"fmt"
	"strings"
)

type QueryRowsScanner func(rows *sql.Rows, throErrorOnFound bool) *Error

type Query struct {
	typ                 TablePrivilege
	requiredValueLength int
	CurrentValues       []any
	placeholderIndex    int

	hasWhere bool
	hasSet   bool

	// For build
	Query         string
	TableRegistry *TableRegistry
	Error         *Error

	// For joins
	tables map[string]*TableRegistry
	fields []string

	RowsScanner      QueryRowsScanner
	throErrorOnFound bool
}
type InnerJoinQuery Query
type InnerJoiner interface {
	On(fieldA, fieldB string) *Query
}

const FIELD_PARSER_PLACEHOLDER = "$$$"

func NewQuery(q string) *Query {
	return &Query{Query: q}
}
func (q *Query) Scan(rows *sql.Rows) *Error {
	return q.RowsScanner(rows, q.throErrorOnFound)
}

// Defines a function to handle returned rows. If no function is passed at all then it doesn't query the returned rows.
func (q *Query) Scanner(fun QueryRowsScanner) *Query {
	q.RowsScanner = fun
	return q
}

// Switch to throw response error on found instead of not found..
func (q *Query) ThroErrorOnFound() *Query {
	q.throErrorOnFound = true
	return q
}
func (q *Query) SeError(e *Error) *Query {
	q.Error = e
	return q
}
func (q *Query) Values(values ...any) *Query {
	if q.Error != nil {
		return q
	}
	if q.typ == SELECT || q.typ == DELETE {
		q.Error = NewError("Must be INSERT | UPDATE ").Status(ErrInvalidMethodChain)
		return q
	}
	var valueAmount = len(values)
	if valueAmount == 0 {
		q.Error = NewError("Cannot use empty values").Status(ErrEmpty)
		return q
	}
	if valueAmount%q.requiredValueLength != 0 {
		q.Error = NewError("Invalid value amount").
			Append(fmt.Sprintf("Wanted: multiple of %d. Recieved: %d", q.requiredValueLength, valueAmount)).
			Status(ErrSyntax)
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

	q.CurrentValues = values
	q.Query += fmt.Sprintf("VALUES %s ", strings.Join(fields, ","))
	return q
}
func (q *Query) Set(field string, value any) *Query {
	if q.Error != nil {
		return q
	}
	if q.typ != UPDATE {
		q.Error = NewError("Must be INSERT or UPDATE").Status(ErrInvalidMethodChain)
		return q
	}
	q.fields = append(q.fields, field)

	if q.hasSet {
		q.Query += ", "
	} else {
		q.hasSet = true
		q.Query += "SET "
	}

	q.Query += fmt.Sprintf("%s = $%d", field, q.placeholderIndex)
	q.placeholderIndex++
	q.CurrentValues = append(q.CurrentValues, value)
	return q
}
func (q *Query) Where(fieldName string, fieldValue any) *Query {
	if q.Error != nil {
		return q
	}
	if q.typ == INSERT {
		q.Error = NewError("Must be INSERT | UPDATE | DELETE").Status(ErrInvalidMethodChain)
		return q
	}
	q.fields = append(q.fields, fieldName)

	if q.hasWhere {
		q.Query += "AND "
	} else {
		q.Query += "WHERE "
		q.hasWhere = true
	}

	q.Query += fmt.Sprintf("%s = $%d ", fieldName, q.placeholderIndex)
	q.placeholderIndex++
	q.CurrentValues = append(q.CurrentValues, fieldValue)
	return q
}
func (q *Query) As(alias string) *Query {
	if q.Error != nil {
		return q
	}
	q.Query += fmt.Sprintf("AS %s ", alias)
	return q
}
func (q *Query) InnerJoin(r *TableRegistry, alias string) *InnerJoinQuery {
	if q.Error != nil {
		return (*InnerJoinQuery)(q)
	}
	q.tables[alias] = r
	q.Query += fmt.Sprintf("INNER JOIN %s AS %s ", r.TableName, alias)
	return (*InnerJoinQuery)(q)
}
func (q *InnerJoinQuery) On(fieldA, fieldB string) *Query {
	if q.Error != nil {
		return (*Query)(q)
	}
	q.Query += fmt.Sprintf("ON %s = %s ", fieldA, fieldB)
	return (*Query)(q)
}
func (q *Query) Returning(fields ...string) *Query {
	if q.Error != nil {
		return q
	}
	if q.typ == SELECT {
		q.Error = NewError("Must be INSERT | UPDATE | DELETE").Status(ErrInvalidMethodChain)
		return q
	}

	q.fields = append(q.fields, fields...)
	q.Query += fmt.Sprintf("RETURNING %s", strings.Join(fields, ", "))
	return q
}

func (q *Query) validateFields() *Error {
	var fields []string
	for _, fieldName := range q.fields {
		var table *TableRegistry = q.TableRegistry

		alias, after, found := strings.Cut(fieldName, ".")
		if found {
			table = q.tables[alias]
			fieldName = after
		}
		_, exists := table.Fields[TableColumnName(fieldName)]
		if !exists {
			return NewError(fmt.Sprintf("%s does not exist in %s", fieldName, q.TableRegistry.TableName)).
				Status(ErrSyntax)
		}
		fields = append(fields, string(fieldName))
	}
	return nil
}
func newQueryOnTable(t *TableRegistry) *Query {
	var q Query
	if t == nil {
		q.Error = NewError("Cannot query nil table").Status(ErrEmpty)
		return &q
	}

	table := (*t.cache)[t.TableName]
	if table.Error != nil {
		return q.SeError(table.Error)
	}
	q.TableRegistry = table
	q.placeholderIndex = 1
	q.tables = make(map[string]*TableRegistry)
	return &q
}
