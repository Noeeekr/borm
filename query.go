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

	Query         string
	TableRegistor *TableRegistor
	Error         *Error

	RowsScanner      QueryRowsScanner
	throErrorOnFound bool
}

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
func (q *Query) Set(field TableColumnName, value any) *Query {
	if q.Error != nil {
		return q
	}
	if q.typ != UPDATE {
		q.Error = NewError("Must be INSERT or UPDATE").Status(ErrInvalidMethodChain)
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
	q.CurrentValues = append(q.CurrentValues, value)
	return q
}
func (q *Query) Where(fieldName TableColumnName, fieldValue any) *Query {
	if q.Error != nil {
		return q
	}
	if q.typ == INSERT {
		q.Error = NewError("Must be INSERT | UPDATE | DELETE").Status(ErrInvalidMethodChain)
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
	q.CurrentValues = append(q.CurrentValues, fieldValue)
	return q
}
func (q *Query) Returning(fields ...TableColumnName) *Query {
	if q.Error != nil {
		return q
	}
	if q.typ == SELECT {
		q.Error = NewError("Must be INSERT | UPDATE | DELETE").Status(ErrInvalidMethodChain)
		return q
	}
	columnsNames, err := q.findFieldsByName(fields...)
	if err != nil {
		q.Error = err
		return q
	}
	q.Query += fmt.Sprintf("RETURNING %s", strings.Join(columnsNames, ", "))
	return q
}

func (q *Query) findFieldsByName(fieldsName ...TableColumnName) ([]string, *Error) {
	var fields []string
	for _, fieldName := range fieldsName {
		_, exists := q.TableRegistor.Fields[fieldName]
		if !exists {
			return nil, NewError(fmt.Sprintf("%s does not exist in %s", fieldName, q.TableRegistor.TableName)).
				Status(ErrNotFound)
		}
		fields = append(fields, string(fieldName))
	}
	return fields, nil
}
func newQueryOnTable(t *TableRegistor) *Query {
	var q Query
	if t == nil {
		q.Error = NewError("Cannot query nil table").Status(ErrEmpty)
		return &q
	}
	table := (*t.cache)[t.TableName]
	if table.Error != nil {
		return q.SeError(table.Error)
	}
	q.TableRegistor = table
	q.placeholderIndex = 1
	return &q
}
