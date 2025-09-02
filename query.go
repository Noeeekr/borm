package borm

import (
	"database/sql"
	"fmt"
	"strings"
)

// ReturnScanner is used by [type Query] Scanner() method,
// it handles the scanning of the returned rows.
//
// On implementation, it must notify if rows were found on returning, the error can be nil.
//
// It will always prioritizes using the error if it exists.
// Otherwise it throws a ErrNotFound if notified that no rows were found. This can be changed with ThrowErrorOnFound()
type ReturnScanner func(rows *sql.Rows) (found bool, err error)

type QueryType int
type Query struct {
	Type                QueryType
	requiredValueLength int
	CurrentValues       []any
	placeholderIndex    int

	// For build
	Query         string
	TableRegistry *TableRegistry
	Error         error
	RegisteredIds map[string]bool

	// map[alias]table
	tableAliases map[string]*TableRegistry
	// string [alias].[fieldname]
	requestedFields []string

	RowsScanner       ReturnScanner
	throwErrorOnFound bool
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

// Unsafe Queries are not stable to use methods are may panic
func NewUnsafeQuery(Type QueryType, q string) *Query {
	return &Query{Type: Type, Query: q}
}
func (q *Query) Scan(rows *sql.Rows) (found bool, err error) {
	return q.RowsScanner(rows)
}

// Scanner expects a function that handles the rows returned by the query.
// If no scanner is present then rows are not scanned.
//
// Scanner throws [type Error ErrNotFound] unless [func ThrowErrorOnFound] is called on this method, in this case it throws an [type Error ErrFound] on the first rows found.
func (q *Query) Scanner(fun ReturnScanner) *Query {
	q.RowsScanner = fun
	return q
}

// Switch to throw response error on found instead of not found..
func (q *Query) ThrowErrorOnFound() *Query {
	q.throwErrorOnFound = true
	return q
}
func (q *Query) SetError(e error) *Query {
	q.Error = e
	return q
}
func (q *Query) Values(values ...any) *Query {
	if q.Error != nil {
		return q
	}
	if q.Type == SELECT || q.Type == DELETE {
		q.Error = ErrorDescription(ErrInvalidMethodChain, "Must be { INSERT, UPDATE }")
		return q
	}
	var valueAmount = len(values)
	if valueAmount == 0 {
		q.Error = ErrorDescription(ErrSyntax, "Values must not be empty. Consider removing it first or handling empty cases.")
		return q
	}
	if valueAmount%q.requiredValueLength != 0 {
		q.Error = ErrorDescription(ErrSyntax, fmt.Sprintf("Invalid value amount. Wanted: multiple of %d. Recieved: %d", q.requiredValueLength, valueAmount))
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
	q.Query += fmt.Sprintf("VALUES %s ", strings.Join(fields, ", "))
	return q
}
func (q *Query) Set(field string, value any) *Query {
	if q.Error != nil {
		return q
	}
	if q.Type != UPDATE {
		q.Error = ErrorDescription(ErrInvalidMethodChain, "Must be INSERT or UPDATE")
		return q
	}
	q.requestedFields = append(q.requestedFields, field)

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
func (q *Query) Where(fieldName string, fieldValues ...any) *Query {
	if q.Error != nil {
		return q
	}
	if q.Type == INSERT {
		q.Error = ErrorDescription(ErrInvalidMethodChain, "Must be INSERT | UPDATE | DELETE")
		return q
	}
	q.registerForValidation(fieldName)

	if q.HasRegisteredID(INTERNAL_WHERE_ID) {
		q.Query += "AND "
	} else {
		q.Query += "WHERE "
		q.RegisterID(INTERNAL_WHERE_ID)
	}

	fieldAmount := len(fieldValues)
	if fieldAmount < 2 {
		q.Query += fmt.Sprintf("%s = $%d ", fieldName, q.placeholderIndex)
		q.placeholderIndex++
	} else {
		// formats to: A in ($1, $2, $3, ...)
		placeholders := make([]string, fieldAmount)
		for i := range fieldValues {
			placeholders[i] = fmt.Sprintf("$%d", q.placeholderIndex)
			q.placeholderIndex++
		}
		q.Query += fmt.Sprintf("%s IN (%s) ", fieldName, strings.Join(placeholders, ", "))
	}
	q.CurrentValues = append(q.CurrentValues, fieldValues...)
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

	// Insert the alias in all anonymous fields
	for i, field := range q.requestedFields {
		if !strings.Contains(field, ".") {
			q.requestedFields[i] = fmt.Sprintf("%s.%s", alias, field)
		}
	}

	// Moves the TableRegistry to the alias
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
	if q.Type == SELECT {
		q.Error = ErrorDescription(ErrInvalidMethodChain, "Must be INSERT | UPDATE | DELETE")
		return q
	}

	q.requestedFields = append(q.requestedFields, fields...)
	q.Query += fmt.Sprintf("RETURNING %s ", strings.Join(fields, ", "))
	return q
}

func (q *Query) validateFields() error {
	var fields []string
	for _, fieldName := range q.requestedFields {
		var table *TableRegistry = q.TableRegistry

		// alias, fieldname
		alias, after, found := strings.Cut(fieldName, ".")
		if found {
			// For join conditions or operations with aliases
			table = q.tableAliases[alias]
			fieldName = after
		} else {
			// For operations without aliases
			table = q.tableAliases[""]
			fieldName = alias
		}
		if table == nil {
			return ErrorDescription(ErrSyntax, fmt.Sprintf("Failed to resolve field [%s]. Perhaps a missing alias", fieldName))
		}
		_, exists := table.Fields[TableFieldName(fieldName)]
		if !exists {
			return ErrorDescription(ErrSyntax, fmt.Sprintf("%s does not exist in %s", fieldName, table.TableName))
		}
		fields = append(fields, string(fieldName))
	}
	return nil
}
func (q *Query) registerForValidation(fieldNames ...string) {
	q.requestedFields = append(q.requestedFields, fieldNames...)
}
func newQueryOnTable(t *TableRegistry) *Query {
	var q Query
	if t == nil {
		q.Error = ErrorDescription(ErrUnexpected, "Unable to query <nil> table.")
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
