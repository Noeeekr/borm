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
	Blocks []string
	Error  error

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

type PartialInnerJoinQuery struct {
	innerQuery *Query
}
type PartialWhereQuery struct {
	innerQuery *Query
}
type AditionalWhereQuery struct {
	*Query
}

// Used internally to identify if a query already has one of these
type BuildStep int

type InternalBitwiseOperator string

const (
	INTERNAL_OPERATOR_OR  InternalBitwiseOperator = "OR"
	INTERNAL_OPERATOR_AND InternalBitwiseOperator = "AND"
)

const (
	INTERNAL_WHERE_ID BuildStep = iota
	INTERNAL_COMPOSED_WHERE_ID
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

	/*
	 */
	// Creates placeholders
	valueBlock := make([]string, valueAmount/q.requiredValueLength)
	valuesIndex := 0
	for i := range valueBlock {
		partialValueBlock := make([]string, q.requiredValueLength)
		for j := range partialValueBlock {
			partialValueBlock[j] = q.usePlaceholder(values[valuesIndex])
			valuesIndex++
		}
		// formats to (a, b, c, ...)
		valueBlock[i] = fmt.Sprintf("(%s)", strings.Join(partialValueBlock, ", "))
	}

	// formats to Values (a, b, c, ...) (e, f, g, ...) ...
	valuesBlock := fmt.Sprintf("VALUES %s", strings.Join(valueBlock, ", "))
	q.appendQueryBlock(valuesBlock)
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
		q.appendQueryBlock(",")
	} else {
		q.setCurrentBuildStep(INTERNAL_SET_ID)
		q.appendQueryBlock("SET")
	}

	q.appendQueryBlock(fmt.Sprintf("%s = %s", field, q.usePlaceholder(value)))
	return q
}
func (q *Query) Where(fieldName string) *PartialWhereQuery {
	return q.where(INTERNAL_OPERATOR_AND, fieldName)
}
func (p *PartialWhereQuery) Equals(fieldValue any) *AditionalWhereQuery {
	if p.innerQuery.Error != nil {
		return &AditionalWhereQuery{
			Query: p.innerQuery,
		}
	}

	p.innerQuery.replaceCurrentQueryBlock(
		fmt.Sprintf(
			"%s = %s",
			p.innerQuery.getCurrentQueryBlock(),
			p.innerQuery.usePlaceholder(fieldValue),
		),
	)
	return &AditionalWhereQuery{
		Query: p.innerQuery,
	}
}
func (q *AditionalWhereQuery) And(fieldName string) *PartialWhereQuery {
	return q.where(INTERNAL_OPERATOR_AND, fieldName)
}
func (q *AditionalWhereQuery) Or(fieldName string) *PartialWhereQuery {
	return q.where(INTERNAL_OPERATOR_OR, fieldName)
}
func (p *PartialWhereQuery) In(fieldValues ...any) *AditionalWhereQuery {
	if p.innerQuery.Error != nil {
		return &AditionalWhereQuery{
			Query: p.innerQuery,
		}
	}

	fieldAmount := len(fieldValues)
	if fieldAmount == 0 {
		p.innerQuery.Error = ErrorDescription(ErrSyntax, "Where clause shouldn't be empty and can cause unwanted returns. Consider removing it if it is intended.")
		return &AditionalWhereQuery{
			Query: p.innerQuery,
		}
	}

	// formats to: A in ($1, $2, $3, ...)
	placeholders := make([]string, fieldAmount)
	for i := range fieldValues {
		placeholders[i] = fmt.Sprintf("%s", p.innerQuery.usePlaceholder(fieldValues[i]))
	}

	p.innerQuery.replaceCurrentQueryBlock(
		fmt.Sprintf(
			"%s IN (%s)",
			p.innerQuery.getCurrentQueryBlock(),
			strings.Join(placeholders, ", "),
		),
	)

	return &AditionalWhereQuery{
		Query: p.innerQuery,
	}
}
func (p *PartialWhereQuery) Like(regex string, caseSensitive bool) *AditionalWhereQuery {
	if p.innerQuery.Error != nil {
		return &AditionalWhereQuery{
			Query: p.innerQuery,
		}
	}

	likeBlock := "LIKE '" + regex + "'"
	if !caseSensitive {
		likeBlock = "I" + likeBlock
	}

	p.innerQuery.replaceCurrentQueryBlock(
		fmt.Sprintf("%s %s",
			p.innerQuery.getCurrentQueryBlock(),
			likeBlock,
		),
	)
	return &AditionalWhereQuery{
		Query: p.innerQuery,
	}
}

/*
Perfoms composed where like:

	WHERE (field = value, field2 IN (value2, value3, value4))

First parameter is the where clause of the query that must be executed in composed
*/
func (q *Query) Compose(qr *AditionalWhereQuery) *AditionalWhereQuery {
	q.replaceCurrentQueryBlock(fmt.Sprintf("(%s)", q.getCurrentQueryBlock()))
	return qr
}
func (q *Query) OrderAscending(fieldName string) *Query {
	if q.Error != nil {
		return q
	}

	q.registerForValidation(fieldName)
	if q.containsBuildStep(INTERNAL_ORDER_ID) {
		q.appendQueryBlock(fmt.Sprintf(", %s ASC", fieldName))
	} else {
		q.setCurrentBuildStep(INTERNAL_ORDER_ID)
		q.appendQueryBlock(fmt.Sprintf("ORDER BY %s ASC", fieldName))
	}

	return q
}
func (q *Query) OrderDescending(fieldName string) *Query {
	if q.Error != nil {
		return q
	}

	q.registerForValidation(fieldName)
	if q.containsBuildStep(INTERNAL_ORDER_ID) {
		q.appendQueryBlock(fmt.Sprintf(", %s DESC", fieldName))
	} else {
		q.setCurrentBuildStep(INTERNAL_ORDER_ID)
		q.appendQueryBlock(fmt.Sprintf("ORDER BY %s DESC", fieldName))
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

	q.appendQueryBlock(fmt.Sprintf("AS %s", alias))
	return q
}
func (q *Query) InnerJoin(r *TableRegistry, alias string) *PartialInnerJoinQuery {
	if q.Error != nil {
		return &PartialInnerJoinQuery{
			innerQuery: q,
		}
	}
	q.setCurrentBuildStep(INTERNAL_JOIN_ID)
	q.tableAliases[alias] = r
	q.appendQueryBlock(fmt.Sprintf("INNER JOIN %s AS %s", r.TableName, alias))
	return &PartialInnerJoinQuery{
		innerQuery: q,
	}
}
func (q *PartialInnerJoinQuery) On(fieldA, fieldB string) *Query {
	if q.innerQuery.Error != nil {
		return q.innerQuery
	}
	q.innerQuery.appendQueryBlock(fmt.Sprintf("ON %s = %s", fieldA, fieldB))
	return q.innerQuery
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
	q.appendQueryBlock(fmt.Sprintf("RETURNING %s", strings.Join(fields, ", ")))
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

	q.appendQueryBlock(fmt.Sprintf("OFFSET %d", amount))
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

	q.appendQueryBlock(fmt.Sprintf("LIMIT %d", amount))
	return q
}
func (q *Query) where(operator InternalBitwiseOperator, fieldName string) *PartialWhereQuery {
	if q.Error != nil {
		return &PartialWhereQuery{
			innerQuery: q,
		}
	}
	if q.Type == INSERT {
		q.Error = ErrorDescription(ErrInvalidMethodChain, "Must be SELECT | UPDATE | DELETE")
		return &PartialWhereQuery{
			innerQuery: q,
		}
	}

	if q.containsBuildStep(INTERNAL_WHERE_ID) {
		if q.getCurrentBuildStep() != INTERNAL_WHERE_ID {
			q.Error = ErrorDescription(ErrSyntax, "WHERE clause must be the current build step.")
			return &PartialWhereQuery{
				q,
			}
		}
		q.replaceCurrentQueryBlock(fmt.Sprintf("%s %s %s", q.getCurrentQueryBlock(), operator, fieldName))
	} else {
		// Where(name).Equals(valor).And(name2).Equals(valor2) => translates to "nome1 = valor1 AND nome2 = valor2"
		// Where(name).Equals(valor).Or(name2).Equals(valor2)  => translates to "nome1 = valor1 OR  nome2 = valor2"
		// Compose(q.Where(name).Equals(valor).And(name2).Equals(valor2))
		//[WHERE][valor] e [AND][valor]  =>  [WHERE][(nome EQUALS valor][AND nome2 EQUALS Valor2)]
		q.appendQueryBlock("WHERE")
		q.appendQueryBlock(fieldName)
		q.setCurrentBuildStep(INTERNAL_WHERE_ID)
	}

	q.registerForValidation(fieldName)
	return &PartialWhereQuery{
		q,
	}
}
func (q *Query) replaceCurrentQueryBlock(query string) {
	if len(q.Blocks) == 0 {
		q.Blocks = append(q.Blocks, query)
	}
	q.Blocks[len(q.Blocks)-1] = query
}
func (q *Query) appendQueryBlock(query string) {
	q.Blocks = append(q.Blocks, query)
}
func (q *Query) getCurrentQueryBlock() string {
	if len(q.Blocks) == 0 {
		return ""
	}
	// Removes the white space added by default
	return q.Blocks[len(q.Blocks)-1]
}
func (q *Query) usePlaceholder(value any) string {
	placeholder := fmt.Sprintf("$%d", q.placeholderIndex)
	q.placeholderIndex++
	q.CurrentValues = append(q.CurrentValues, value)
	return placeholder
}
func (q *Query) build() string {
	if Settings().Environment().GetEnvironment() == DEBUGGING {
		fmt.Printf("[%s]\n", strings.Join(q.Blocks, "]["))
	}
	return strings.Join(q.Blocks, " ")
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
func (q *QueryValidator) isValid() error {
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
	q.appendQueryBlock(str)
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
