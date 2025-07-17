package registers

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/Noeeekr/borm/common"
)

type QueryRowsScanner func(rows *sql.Rows, throwErrorOnFound bool) *common.Error

type Query struct {
	typ                 TablePrivilege
	requiredValueLength int
	CurrentValues       []any
	placeholderIndex    int

	hasWhere bool
	hasSet   bool

	Query       string
	Information *Table
	Error       *common.Error

	RowsScanner       QueryRowsScanner
	throwErrorOnFound bool
}

// This ******* string becomes a ******** of complex settings just to become a ******* string again at the end, funny isnt it?
func NewQuery(q string) *Query {
	return &Query{Query: q}
}
func (q *Query) Scan(rows *sql.Rows) *common.Error {
	return q.RowsScanner(rows, q.throwErrorOnFound)
}

// Defines a function to handle returned rows. If no function is passed at all then it doesn't query the returned rows.
func (q *Query) Scanner(fun QueryRowsScanner) *Query {
	q.RowsScanner = fun
	return q
}

// Switch to throw response error on found instead of not found..
func (q *Query) ThrowErrorOnFound() *Query {
	q.throwErrorOnFound = true
	return q
}
func (q *Query) SetError(e *common.Error) *Query {
	q.Error = e
	return q
}
func (q *Query) Values(values ...any) *Query {
	if q.Error != nil {
		return q
	}
	if q.typ == SELECT || q.typ == DELETE {
		q.Error = common.NewError("Must be INSERT | UPDATE ").Status(common.ErrInvalidMethodChain)
		return q
	}
	var valueAmount = len(values)
	if valueAmount == 0 {
		q.Error = common.NewError("Cannot use empty values").Status(common.ErrEmpty)
		return q
	}
	if valueAmount%q.requiredValueLength != 0 {
		q.Error = common.NewError("Invalid value amount").
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

	q.CurrentValues = values
	q.Query += fmt.Sprintf("VALUES %s ", strings.Join(fields, ","))
	return q
}
func (q *Query) Set(field TableColumnName, value any) *Query {
	if q.Error != nil {
		return q
	}
	if q.typ != UPDATE {
		q.Error = common.NewError("Must be INSERT or UPDATE").Status(common.ErrInvalidMethodChain)
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
		q.Error = common.NewError("Must be INSERT | UPDATE | DELETE").Status(common.ErrInvalidMethodChain)
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
		q.Error = common.NewError("Must be INSERT | UPDATE | DELETE").Status(common.ErrInvalidMethodChain)
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

func (q *Query) findFieldsByName(fieldsName ...TableColumnName) ([]string, *common.Error) {
	var fields []string
	for _, fieldName := range fieldsName {
		_, exists := q.Information.Fields[fieldName]
		if !exists {
			return nil, common.NewError(fmt.Sprintf("%s does not exist in %s", fieldName, q.Information.TableName)).
				Status(common.ErrNotFound)
		}
		fields = append(fields, string(fieldName))
	}
	return fields, nil
}
func newQueryOnTable(t *Table) *Query {
	var q Query
	if t == nil {
		q.Error = common.NewError("Cannot query nil table").Status(common.ErrEmpty)
		return &q
	}
	table := (*t.cache)[t.TableName]
	if table.Error != nil {
		return q.SetError(table.Error)
	}
	q.Information = table
	q.placeholderIndex = 1
	return &q
}
