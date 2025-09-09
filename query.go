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
	Query string
	Error error

	// For build validation
	*QueryValidator

	RowsScanner       ReturnScanner
	throwErrorOnFound bool
}

type QueryValidator struct {
	// Pointer to the table this query is being built on
	TableRegistry *TableRegistry

	tableAliases    map[string]*TableRegistry
	requestedFields []string

	RegisteredBuildSteps map[BuildStep]bool
	CurrentBuildStep     BuildStep
}

type WhereQuery Query
type InnerJoinQuery Query
type InnerJoiner interface {
	On(fieldA, fieldB string) *Query
}

// Used internally to identify if a query already has one of these
type BuildStep int

const (
	INTERNAL_WHERE_ID BuildStep = iota
	INTERNAL_SET_ID
	INTERNAL_ORDER_ID
	INTERNAL_JOIN_ID
)

const FIELD_PARSER_PLACEHOLDER = "$$$"

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

	if q.containsBuildStep(INTERNAL_SET_ID) {
		q.Query += ", "
	} else {
		q.setCurrentBuildStep(INTERNAL_SET_ID)
		q.Query += "SET "
	}

	q.Query += fmt.Sprintf("%s = $%d", field, q.placeholderIndex)
	q.placeholderIndex++
	q.CurrentValues = append(q.CurrentValues, value)
	return q
}
func (q *Query) Where(fieldName string) *WhereQuery {
	if q.Error != nil {
		return (*WhereQuery)(q)
	}
	if q.Type == INSERT {
		q.Error = ErrorDescription(ErrInvalidMethodChain, "Must be SELECT | UPDATE | DELETE")
		return (*WhereQuery)(q)
	}

	if q.containsBuildStep(INTERNAL_WHERE_ID) {
		if q.getCurrentBuildStep() != INTERNAL_WHERE_ID {
			q.Error = ErrorDescription(ErrSyntax, "WHERE clause must be the current build step.")
			return (*WhereQuery)(q)
		}
		q.Query += fmt.Sprintf("AND %s ", fieldName)
	} else {
		q.Query += fmt.Sprintf("WHERE %s ", fieldName)
		q.setCurrentBuildStep(INTERNAL_WHERE_ID)
	}

	q.registerForValidation(fieldName)
	return (*WhereQuery)(q)
}
func (q *WhereQuery) Equals(fieldValue any) *Query {
	if q.Error != nil {
		return (*Query)(q)
	}

	q.Query += fmt.Sprintf("= $%d ", q.placeholderIndex)
	q.placeholderIndex++

	q.CurrentValues = append(q.CurrentValues, fieldValue)
	return (*Query)(q)
}
func (q *WhereQuery) In(fieldValues ...any) *Query {
	if q.Error != nil {
		return (*Query)(q)
	}

	fieldAmount := len(fieldValues)
	if fieldAmount == 0 {
		q.Error = ErrorDescription(ErrSyntax, "Where clause shouldn't be empty and can cause unwanted returns. Consider removing it if it is intended.")
		return (*Query)(q)
	}

	// formats to: A in ($1, $2, $3, ...)
	placeholders := make([]string, fieldAmount)
	for i := range fieldValues {
		placeholders[i] = fmt.Sprintf("$%d", q.placeholderIndex)
		q.placeholderIndex++
	}
	q.Query += fmt.Sprintf("IN (%s) ", strings.Join(placeholders, ", "))

	q.CurrentValues = append(q.CurrentValues, fieldValues...)

	return (*Query)(q)
}
func (q *WhereQuery) Like(regex string, caseSensitive bool) *Query {
	if q.Error != nil {
		return (*Query)(q)
	}

	if !caseSensitive {
		q.Query += "I"
	}
	q.Query += "LIKE '" + regex + "' "
	q.placeholderIndex++
	return (*Query)(q)
}
func (q *Query) OrderAscending(fieldName string) *Query {
	if q.Error != nil {
		return q
	}

	q.registerForValidation(fieldName)
	if q.containsBuildStep(INTERNAL_ORDER_ID) {
		q.Query += fmt.Sprintf(", %s ASC", fieldName)
	} else {
		q.setCurrentBuildStep(INTERNAL_ORDER_ID)
		q.Query += fmt.Sprintf("ORDER BY %s ASC ", fieldName)
	}

	return q
}
func (q *Query) OrderDescending(fieldName string) *Query {
	if q.Error != nil {
		return q
	}

	q.registerForValidation(fieldName)
	if q.containsBuildStep(INTERNAL_ORDER_ID) {
		q.Query += fmt.Sprintf(", %s DESC", fieldName)
	} else {
		q.setCurrentBuildStep(INTERNAL_ORDER_ID)
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
	q.setCurrentBuildStep(INTERNAL_JOIN_ID)
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

func (q *Query) Offset(amount int) *Query {
	if q.Error != nil {
		return q
	}
	if q.Type != SELECT {
		q.Error = ErrorDescription(ErrInvalidMethodChain, "Must be SELECT")
		return q
	}

	q.Query += fmt.Sprintf("OFFSET %d ", amount)
	return q
}
func (q *Query) Limit(amount int) *Query {
	if q.Error != nil {
		return q
	}
	if q.Type != SELECT {
		q.Error = ErrorDescription(ErrInvalidMethodChain, "Must be SELECT")
		return q
	}

	q.Query += fmt.Sprintf("LIMIT %d ", amount)
	return q
}
func (q *Query) Like(regex string, caseSensitive bool) *Query {
	if q.Error != nil {
		return q
	}
	if q.Type == INSERT {
		q.Error = ErrorDescription(ErrInvalidMethodChain, "Must be SELECT | UPDATE | DELETE")
	}
	if q.getCurrentBuildStep() != INTERNAL_WHERE_ID {
		q.Error = ErrorDescription(ErrSyntax, "LIKE must be used after a WHERE clause")
		return q
	}

	if caseSensitive {
		q.Query += "I"
	}
	q.Query += fmt.Sprintf("LIKE %s ", q.getCurrentPlaceholder(regex))
	return q
}

func (q *Query) getCurrentPlaceholder(value any) string {
	placeholder := fmt.Sprintf("$%d", q.placeholderIndex)
	q.placeholderIndex++
	q.CurrentValues = append(q.CurrentValues, value)
	return placeholder
}
func (q *QueryValidator) containsBuildStep(step BuildStep) bool {
	_, found := q.RegisteredBuildSteps[step]
	return found
}
func (q *QueryValidator) setCurrentBuildStep(step BuildStep) {
	q.RegisteredBuildSteps[step] = true
	q.CurrentBuildStep = step
}
func (q *QueryValidator) getCurrentBuildStep() BuildStep {
	return q.CurrentBuildStep
}
func (q *QueryValidator) validateFields() error {
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
func (q *QueryValidator) registerForValidation(fieldNames ...string) {
	q.requestedFields = append(q.requestedFields, fieldNames...)
}
func newQueryValidator(t *TableRegistry) *QueryValidator {
	return &QueryValidator{
		RegisteredBuildSteps: make(map[BuildStep]bool),
		CurrentBuildStep:     -1,

		requestedFields: make([]string, 0),
		tableAliases:    make(map[string]*TableRegistry),

		TableRegistry: t,
	}
}

// Unsafe Queries doesn't need a table, and are not stable to use methods are may panic.
func newUnsafeQuery(typ QueryType, str string) *Query {
	q := Query{}
	q.QueryValidator = newQueryValidator(nil)
	q.Type = typ
	q.placeholderIndex = 1
	q.Query = str
	return &q
}
func NewQuery(t *TableRegistry, typ QueryType) *Query {
	var q Query
	if t == nil {
		q.Error = ErrorDescription(ErrUnexpected, "Unable to query <nil> table.")
		return &q
	}

	table := (*t.databaseCache)[t.TableName]
	if table.Error != nil {
		return q.SetError(table.Error)
	}

	q.placeholderIndex = 1
	q.QueryValidator = newQueryValidator(t)

	return &q
}
