package registers

import (
	"fmt"
	"strings"

	"github.com/Noeeekr/borm/common"
)

type QueryType int

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
		q.Error = common.NewError("Must be INSERT or UPDATE").Status(common.ErrInvalidMethodChain)
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

func (q *Query) findFieldsByName(fieldsName ...TableColumnName) ([]string, *common.Error) {
	var fields []string
	for _, fieldName := range fieldsName {
		_, exists := q.Information.Fields[fieldName]
		if !exists {
			return nil, common.NewError(fmt.Sprintf("%s does not exist in %s", fieldName, q.Information.Name)).
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
	table := (*t.cache)[t.Name]
	if table.Error != nil {
		return q.SetError(table.Error)
	}
	q.Information = table
	q.placeholderIndex = 1
	return &q
}
