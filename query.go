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

	// For build
	Query         string
	TableRegistry *TableRegistry
	Error         *Error
	RegisteredIds map[string]bool

	// map[alias]table
	tableAliases map[string]*TableRegistry
	// string [alias].[fieldname]
	fields []string

	RowsScanner      QueryRowsScanner
	throErrorOnFound bool
}
type OrderChain struct {
	*Query
}

type InnerJoinQuery Query
type InnerJoiner interface {
	On(fieldA, fieldB string) *Query
}

// Used internally to identify if a chain already has one of these
const INTERNAL_WHERE_ID = "where"
const INTERNAL_SET_ID = "set"
const INTERNAL_ORDER_ID = "id"
const INTERNAL_JOIN_ID = "join"

const FIELD_PARSER_PLACEHOLDER = "$$$"

func NewQuery(q string) *Query {
	return &Query{Query: q}
}
func (q *Query) Scan(rows *sql.Rows) *Error {
	return q.RowsScanner(rows, q.throErrorOnFound)
}

// Scanner expects a function that handles the rows returned by the query.
// If no scanner is present then rows are not scanned.
//
// Scanner throws [type Error ErrNotFound] unless [func ThrowErrorOnFound] is called on this method, in this case it throws an [type Error ErrFound] on the first rows found.
func (q *Query) Scanner(fun QueryRowsScanner) *Query {
	q.RowsScanner = fun
	return q
}

// Switch to throw response error on found instead of not found..
func (q *Query) ThrowErrorOnFound() *Query {
	q.throErrorOnFound = true
	return q
}
func (q *Query) SetError(e *Error) *Query {
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

	if q.HasRegisteredID(INTERNAL_SET_ID) {
		q.Query += ", "
	} else {
		q.RegisterID(INTERNAL_SET_ID)
		q.Query += "SET "
	}

	q.Query += fmt.Sprintf("%s = $%d", field, q.placeholderIndex)
	q.placeholderIndex++
	q.CurrentValues = append(q.CurrentValues, value)
	return q
}
func (q *Query) RegisterID(id string) {
	q.RegisteredIds[id] = true
}
func (q *Query) HasRegisteredID(id string) bool {
	_, found := q.RegisteredIds[id]
	return found
}
func (q *Query) Where(fieldName string, fieldValue any) *Query {
	if q.Error != nil {
		return q
	}
	if q.typ == INSERT {
		q.Error = NewError("Must be INSERT | UPDATE | DELETE").Status(ErrInvalidMethodChain)
		return q
	}
	q.registerForValidation(fieldName)

	if q.HasRegisteredID(INTERNAL_WHERE_ID) {
		q.Query += "AND "
	} else {
		q.Query += "WHERE "
		q.RegisterID(INTERNAL_WHERE_ID)
	}

	q.Query += fmt.Sprintf("%s = $%d ", fieldName, q.placeholderIndex)
	q.placeholderIndex++
	q.CurrentValues = append(q.CurrentValues, fieldValue)
	return q
}
func (q *Query) OrderAscending(fieldName string) *Query {
	if q.Error != nil {
		return q
	}

	q.registerForValidation(fieldName)
	if q.HasRegisteredID(INTERNAL_ORDER_ID) {
		q.Query += fmt.Sprintf(", %s ASC", fieldName)
	} else {
		q.RegisterID(INTERNAL_ORDER_ID)
		q.Query += fmt.Sprintf("ORDER BY %s ASC ", fieldName)
	}

	return q
}
func (q *Query) OrderDescending(fieldName string) *Query {
	if q.Error != nil {
		return q
	}

	q.registerForValidation(fieldName)
	if q.HasRegisteredID(INTERNAL_ORDER_ID) {
		q.Query += fmt.Sprintf(", %s DESC", fieldName)
	} else {
		q.RegisterID(INTERNAL_ORDER_ID)
		q.Query += fmt.Sprintf("ORDER BY %s DESC ", fieldName)
	}

	return q
}
func (q *Query) As(alias string) *Query {
	if q.Error != nil {
		return q
	}

aliasChecker:
	for i, field := range q.fields {
		for _, char := range field {
			if char == '.' {
				continue aliasChecker
			}
		}
		q.fields[i] = fmt.Sprintf("%s.%s", alias, field)
	}
	q.tableAliases[alias] = q.tableAliases[""]
	delete(q.tableAliases, "")

	q.Query += fmt.Sprintf("AS %s ", alias)
	return q
}
func (q *Query) InnerJoin(r *TableRegistry, alias string) *InnerJoinQuery {
	if q.Error != nil {
		return (*InnerJoinQuery)(q)
	}
	q.RegisterID(INTERNAL_JOIN_ID)
	q.tableAliases[alias] = r
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

		// alias, fieldname
		alias, after, found := strings.Cut(fieldName, ".")
		if found {
			// For join conditions or SELECT() with aliases
			table = q.tableAliases[alias]
			fieldName = after
		} else {
			// For SELECT() without aliases
			table = q.tableAliases[""]
			fieldName = alias
		}
		if table == nil {
			return NewError(fmt.Sprintf("Failed to resolve field [%s]. Perhaps an missing alias", fieldName)).
				Status(ErrSyntax)
		}
		_, exists := table.Fields[TableColumnName(fieldName)]
		if !exists {
			return NewError(fmt.Sprintf("%s does not exist in %s", fieldName, table.TableName)).
				Status(ErrSyntax)
		}
		fields = append(fields, string(fieldName))
	}
	return nil
}
func (q *Query) registerForValidation(fieldNames ...string) {
	q.fields = append(q.fields, fieldNames...)
}
func newQueryOnTable(t *TableRegistry) *Query {
	var q Query
	if t == nil {
		q.Error = NewError("Cannot query nil table").Status(ErrEmpty)
		return &q
	}

	table := (*t.databaseCache)[t.TableName]
	if table.Error != nil {
		return q.SetError(table.Error)
	}
	q.TableRegistry = table
	q.placeholderIndex = 1
	q.tableAliases = make(map[string]*TableRegistry)
	q.RegisteredIds = make(map[string]bool)
	return &q
}
