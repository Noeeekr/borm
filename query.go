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

type QueryBlock struct {
	Block     string
	BlockType BuildStep
}
type QueryType int
type Query struct {
	Type                QueryType
	requiredValueLength int
	CurrentValues       []any
	placeholderIndex    int

	// For build
	Blocks []QueryBlock
	Error  error

	// For build validation
	*QueryValidator

	RowsScanner       ReturnScanner
	throwErrorOnFound bool
}

type RequiredQuery struct {
	innerQuery *Query
}
type OptionalQuery struct {
	*Query
}

type QueryValidator struct {
	// Pointer to the table this query is being built on
	TableRegistry *TableRegistry

	tableAliases    map[string]*TableRegistry
	requestedFields []string

	RegisteredBuildSteps map[BuildStep]int
	CurrentBuildStep     BuildStep
}

type ConditionalQuery struct {
	parentQuery *Query
	block       string
	error       error
}
type PartialInnerJoinQuery RequiredQuery
type PartialWhereQuery RequiredQuery
type AditionalWhereQuery OptionalQuery

// Used internally to identify if a query already has one of these
type BuildStep int

type InternalBitwiseOperator string

const (
	INTERNAL_OPERATOR_NONE InternalBitwiseOperator = ""
	INTERNAL_OPERATOR_OR   InternalBitwiseOperator = "OR"
	INTERNAL_OPERATOR_AND  InternalBitwiseOperator = "AND"
)

const (
	INTERNAL_WHERE_ID BuildStep = iota
	INTERNAL_COMPOSED_WHERE_ID
	INTERNAL_SET_ID
	INTERNAL_ORDER_ID
	INTERNAL_JOIN_ID
)

const FIELD_PARSER_PLACEHOLDER = "$$$"

func newConditionalQuery(parent *Query, block string, error error) *ConditionalQuery {
	return &ConditionalQuery{
		parentQuery: parent,
		block:       block,
		error:       error,
	}
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
func (q *Query) Where(conditional *ConditionalQuery) *Query {
	return q.where(INTERNAL_OPERATOR_NONE, conditional).parentQuery
}
func (q *Query) And(conditionals ...*ConditionalQuery) *ConditionalQuery {
	return q.where(INTERNAL_OPERATOR_AND, conditionals...)
}
func (q *Query) Or(conditionals ...*ConditionalQuery) *ConditionalQuery {
	return q.where(INTERNAL_OPERATOR_OR, conditionals...)
}
func (p *ConditionalQuery) IsIn(fieldValues ...any) *ConditionalQuery {
	if p.error != nil {
		return p
	}

	fieldAmount := len(fieldValues)
	if fieldAmount == 0 {
		p.error = ErrorDescription(ErrSyntax, "Where clause shouldn't be empty and can cause unwanted returns. Consider removing it if it is intended.")
		return p
	}

	// formats to: A in ($1, $2, $3, ...)
	placeholders := make([]string, fieldAmount)
	for i := range fieldValues {
		placeholders[i] = p.parentQuery.usePlaceholder(fieldValues[i])
	}

	p.block += fmt.Sprintf("IN (%s) ", strings.Join(placeholders, ", "))
	return p
}
func (p *ConditionalQuery) IsLessThan(fieldValue any) *ConditionalQuery {
	if p.error != nil {
		return p
	}

	p.block += "> " + p.parentQuery.usePlaceholder(fieldValue)
	return p
}
func (p *ConditionalQuery) IsBiggerThan(fieldValue any) *ConditionalQuery {
	if p.error != nil {
		return p
	}

	p.block += "< " + p.parentQuery.usePlaceholder(fieldValue)
	return p
}
func (p *ConditionalQuery) IsAfter(fieldValue any) *ConditionalQuery {
	return p.IsBiggerThan(fieldValue)
}
func (p *ConditionalQuery) IsBefore(fieldValue any) *ConditionalQuery {
	return p.IsLessThan(fieldValue)
}
func (p *ConditionalQuery) IsEqual(fieldValue any) *ConditionalQuery {
	if p.error != nil {
		return p
	}

	if fieldValue == nil {
		p.block += "IS NULL "
		return p
	}
	p.block += "= " + p.parentQuery.usePlaceholder(fieldValue)
	return p
}
func (p *ConditionalQuery) IsInRange(fieldValueA, fieldValueB any) *ConditionalQuery {
	if p.error != nil {
		return p
	}

	p.block += fmt.Sprintf(
		"BETWEEN %s AND %s ",
		p.parentQuery.usePlaceholder(fieldValueA),
		p.parentQuery.usePlaceholder(fieldValueB),
	)
	return p
}
func (p *ConditionalQuery) IsLike(regex string, caseSensitive bool) *ConditionalQuery {
	if p.error != nil {
		return p
	}
	likeBlock := "LIKE '" + regex + "'"
	if !caseSensitive {
		likeBlock = "I" + likeBlock
	}
	p.block += likeBlock + " "
	return p
}

func (q *Query) Compose(conditional *ConditionalQuery) *ConditionalQuery {
	return newConditionalQuery(q, fmt.Sprintf("(%s)", conditional.block), nil)
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
func (q *Query) RightJoin(r *TableRegistry, alias string) *PartialInnerJoinQuery {
	return q.join(r, "RIGHT JOIN", alias)
}
func (q *Query) CrossJoin(r *TableRegistry, alias string) *PartialInnerJoinQuery {
	return q.join(r, "CROSS JOIN", alias)
}
func (q *Query) Join(r *TableRegistry, alias string) *PartialInnerJoinQuery {
	return q.join(r, "JOIN", alias)
}
func (q *Query) LeftJoin(r *TableRegistry, alias string) *PartialInnerJoinQuery {
	return q.join(r, "LEFT JOIN", alias)
}
func (q *Query) InnerJoin(r *TableRegistry, alias string) *PartialInnerJoinQuery {
	return q.join(r, "INNER JOIN", alias)
}
func (q *Query) join(r *TableRegistry, joinType, alias string) *PartialInnerJoinQuery {
	if q.Error != nil {
		return &PartialInnerJoinQuery{
			innerQuery: q,
		}
	}
	q.setCurrentBuildStep(INTERNAL_JOIN_ID)
	q.tableAliases[alias] = r
	q.appendQueryBlock(fmt.Sprintf("%s %s AS %s", joinType, r.TableName, alias))
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

func (q *Query) Field(fieldName string) *ConditionalQuery {
	if q.Error != nil {
		return newConditionalQuery(q, fieldName, q.Error)
	}
	if q.Type == INSERT {
		q.Error = ErrorDescription(ErrInvalidMethodChain, "Must be SELECT | UPDATE | DELETE")
		return newConditionalQuery(q, fieldName, q.Error)
	}

	q.registerForValidation(fieldName)
	q.setCurrentBuildStep(INTERNAL_WHERE_ID)
	return newConditionalQuery(q, fieldName+" ", q.Error)
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

// On first use where appends the WHERE clause, after that it can be selectively used to append the operator and fields
// First parameter specifies the operator to be used to append with the previous where rule if exists.
// Second parameter is the fieldName that the rule will validate into.
// Third parameter tells if the query should merge to the previous one or creating new block, if true the operator will be used for that reason.
func (q *Query) where(operator InternalBitwiseOperator, conditionals ...*ConditionalQuery) *ConditionalQuery {
	if q.Error != nil {
		return newConditionalQuery(q, "", q.Error)
	}
	if q.Type == INSERT {
		return newConditionalQuery(q, "", ErrorDescription(ErrInvalidMethodChain, "Must be SELECT | UPDATE | DELETE"))
	}
	if len(conditionals) == 0 {
		return newConditionalQuery(q, "", ErrorDescription(ErrSyntax, "Conditionals shouldn't be used with empty values"))
	}

	switch operator {
	case INTERNAL_OPERATOR_NONE:
		q.appendQueryBlock("WHERE " + conditionals[0].block)
		return newConditionalQuery(q, "", nil)
	case INTERNAL_OPERATOR_OR:
		blocks := make([]string, len(conditionals))
		for i, conditional := range conditionals {
			blocks[i] = conditional.block
		}
		return newConditionalQuery(q, strings.Join(blocks, " OR "), nil)
	case INTERNAL_OPERATOR_AND:
		blocks := make([]string, len(conditionals))
		for i, conditional := range conditionals {
			blocks[i] = conditional.block
		}
		return newConditionalQuery(q, strings.Join(blocks, " AND "), nil)
	default:
		return newConditionalQuery(q, "", ErrorDescription(ErrUnexpected, "How????"))
	}
}

//	func (q *Query) getCurrentQueryBlockIndex() int {
//		return len(q.Blocks) - 1
//	}
//
//	func (q *Query) removeCurrentQueryBlock() QueryBlock {
//		if len(q.Blocks) == 0 {
//			return QueryBlock{
//				BlockType: -1,
//				Block:     "",
//			}
//		}
//		removedBlock := q.Blocks[len(q.Blocks)-1]
//		q.Blocks = q.Blocks[:len(q.Blocks)-1]
//		return removedBlock
//	}
//
//	func (q *Query) replaceCurrentQueryBlock(query string) {
//		if len(q.Blocks) == 0 {
//			q.Blocks = append(q.Blocks, QueryBlock{Block: query, BlockType: q.getLastBlockType()})
//		}
//		q.Blocks[len(q.Blocks)-1] = QueryBlock{Block: query, BlockType: q.getLastBlockType()}
//	}
func (q *Query) appendQueryBlock(query string) {
	q.Blocks = append(q.Blocks, QueryBlock{Block: query, BlockType: q.getLastBlockType()})
}

//	func (q *Query) getCurrentQueryBlock() QueryBlock {
//		if len(q.Blocks) == 0 {
//			return QueryBlock{
//				BlockType: -1,
//				Block:     "",
//			}
//		}
//		return q.Blocks[len(q.Blocks)-1]
//	}
func (q *Query) usePlaceholder(value any) string {
	placeholder := fmt.Sprintf("$%d", q.placeholderIndex)
	q.placeholderIndex++
	q.CurrentValues = append(q.CurrentValues, value)
	return placeholder
}
func (q *Query) build() string {
	blocks := make([]string, len(q.Blocks))
	for i := range q.Blocks {
		blocks[i] = q.Blocks[i].Block
	}
	if Settings().Environment().GetEnvironment() == DEBUGGING {
		fmt.Printf("[%s]\n", strings.Join(blocks, "]\n["))
	}
	return strings.Join(blocks, " ")
}

//	func (q *QueryValidator) getBuildStepAmount(step BuildStep) int {
//		return q.RegisteredBuildSteps[step]
//	}
func (q *QueryValidator) containsBuildStep(step BuildStep) bool {
	_, found := q.RegisteredBuildSteps[step]
	return found
}
func (q *QueryValidator) setCurrentBuildStep(step BuildStep) {
	q.RegisteredBuildSteps[step] += 1
	q.CurrentBuildStep = step
}
func (q *QueryValidator) getLastBlockType() BuildStep {
	return q.CurrentBuildStep
}
func (q *QueryValidator) isValid() error {
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
	}
	return nil
}
func (q *QueryValidator) registerForValidation(fieldNames ...string) {
	q.requestedFields = append(q.requestedFields, fieldNames...)
}
func newQueryValidator(t *TableRegistry) *QueryValidator {
	return &QueryValidator{
		RegisteredBuildSteps: make(map[BuildStep]int),
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
