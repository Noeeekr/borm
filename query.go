package borm

import (
	"database/sql"
	"errors"
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
	BlockType QueryStep
}
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

type NonOptionalQuery struct {
	parentQuery *Query
}
type OptionalQuery struct {
	*Query
}

type QueryValidator struct {
	// Pointer to the table this query is being built on
	TableRegistry *TableRegistry

	tableAliases   map[string]*TableRegistry
	selectorFields []string

	QuerySteps map[QueryStep]int
	QueryStep  QueryStep
}

type ConditionalQuery struct {
	parentQuery *Query
	block       string
	error       error
}

type QueryStep int
type QueryType int
type InternalBitwiseOperator string
type PartialInnerJoinQuery NonOptionalQuery
type PartialWhereQuery NonOptionalQuery
type AdditionalSelectQuery OptionalQuery
type AdditionalWhereQuery OptionalQuery

// Used internally to identify if a query already has one of these
const (
	INTERNAL_OPERATOR_OR  InternalBitwiseOperator = "OR"
	INTERNAL_OPERATOR_AND InternalBitwiseOperator = "AND"
)

const (
	SELECT QueryType = iota
	UPDATE
	DELETE
	INSERT

	CREATE
	DROP

	ALL
)

const (
	INTERNAL_COMPOSED_WHERE_TOKEN QueryStep = iota
	INTERNAL_GROUP_BY_TOKEN
	INTERNAL_ORDER_TOKEN
	INTERNAL_WHERE_TOKEN
	INTERNAL_JOIN_TOKEN
	INTERNAL_SET_TOKEN
	INTERNAL_AS_TOKEN
)

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
func (q *Query) SetError(e string) *Query {
	q.Error = errors.New(e)
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
	q.selectorFields = append(q.selectorFields, field)

	if q.GetQueryStep(INTERNAL_SET_TOKEN) {
		q.appendQueryBlock(",")
	} else {
		q.SetQueryStep(INTERNAL_SET_TOKEN)
		q.appendQueryBlock("SET")
	}

	q.appendQueryBlock(fmt.Sprintf("%s = %s", field, q.usePlaceholder(value)))
	return q
}
func (q *Query) Where(conditional *ConditionalQuery) *Query {
	if q.Error != nil {
		return q
	}
	if q.Type == INSERT {
		return q.SetError("Must be SELECT | UPDATE | DELETE")
	}
	if conditional != nil {
		q.appendQueryBlock("WHERE " + conditional.block)
	}
	return q
}
func (q *Query) And(conditionals ...*ConditionalQuery) *ConditionalQuery {
	cleanConditionals := []string{}
	for _, conditional := range conditionals {
		if conditional != nil {
			cleanConditionals = append(cleanConditionals, conditional.block)
		}
	}
	if len(cleanConditionals) == 0 {
		return nil
	}
	return newConditionalQuery(q, strings.Join(cleanConditionals, " AND "), nil)
}
func (q *Query) Or(conditionals ...*ConditionalQuery) *ConditionalQuery {
	cleanConditionals := []string{}
	for _, conditional := range conditionals {
		if conditional != nil {
			cleanConditionals = append(cleanConditionals, conditional.block)
		}
	}
	if len(cleanConditionals) == 0 {
		return nil
	}
	return newConditionalQuery(q, strings.Join(cleanConditionals, " OR "), nil)
}
func (p *ConditionalQuery) IsAny(fieldValues ...any) *ConditionalQuery {
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
	if conditional == nil {
		return nil
	}
	q.SetQueryStep(INTERNAL_COMPOSED_WHERE_TOKEN)
	return newConditionalQuery(q, fmt.Sprintf("(%s)", conditional.block), nil)
}
func (q *Query) OrderAscending(fieldName string) *Query {
	if q.Error != nil {
		return q
	}

	q.registerForValidation(fieldName)
	if q.GetQueryStep(INTERNAL_ORDER_TOKEN) {
		q.appendQueryBlock(fmt.Sprintf(", %s ASC", fieldName))
	} else {
		q.SetQueryStep(INTERNAL_ORDER_TOKEN)
		q.appendQueryBlock(fmt.Sprintf("ORDER BY %s ASC", fieldName))
	}

	return q
}
func (q *Query) OrderDescending(fieldName string) *Query {
	if q.Error != nil {
		return q
	}

	q.registerForValidation(fieldName)
	if q.GetQueryStep(INTERNAL_ORDER_TOKEN) {
		q.appendQueryBlock(fmt.Sprintf(", %s DESC", fieldName))
	} else {
		q.SetQueryStep(INTERNAL_ORDER_TOKEN)
		q.appendQueryBlock(fmt.Sprintf("ORDER BY %s DESC", fieldName))
	}

	return q
}
func (q *AdditionalSelectQuery) As(alias string) *Query {
	if q.Error != nil {
		return q.Query
	}

	// Moves the TableRegistry to the alias
	q.tableAliases[alias] = q.tableAliases[""]
	delete(q.tableAliases, "")

	q.SetQueryStep(INTERNAL_AS_TOKEN)
	q.appendQueryBlock(fmt.Sprintf("AS %s", alias))
	return q.Query
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
		return newPartialInnerJoinQuery(q)
	}
	q.SetQueryStep(INTERNAL_JOIN_TOKEN)
	q.tableAliases[alias] = r
	q.appendQueryBlock(fmt.Sprintf("%s %s AS %s", joinType, r.TableName, alias))
	return newPartialInnerJoinQuery(q)
}
func (q *PartialInnerJoinQuery) On(fieldA, fieldB string) *Query {
	if q.parentQuery.Error != nil {
		return q.parentQuery
	}
	q.parentQuery.appendQueryBlock(fmt.Sprintf("ON %s = %s", fieldA, fieldB))
	return q.parentQuery
}
func (q *Query) Returning(fields ...string) *Query {
	if q.Error != nil {
		return q
	}
	if q.Type == SELECT {
		q.Error = ErrorDescription(ErrInvalidMethodChain, "Must be INSERT | UPDATE | DELETE")
		return q
	}

	q.selectorFields = append(q.selectorFields, fields...)
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
	q.SetQueryStep(INTERNAL_WHERE_TOKEN)
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

func (q *Query) GroupBy(fields ...string) *Query {
	q.SetQueryStep(INTERNAL_GROUP_BY_TOKEN)
	q.appendQueryBlock("GROUP BY " + strings.Join(fields, ", "))
	return q
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

//	func (q *QueryValidator) QueryStepAmount(step QueryStep) int {
//		return q.SetQuerySteps[step]
//	}
func (q *QueryValidator) GetQueryStep(step QueryStep) bool {
	_, found := q.QuerySteps[step]
	return found
}
func (q *QueryValidator) SetQueryStep(step QueryStep) {
	q.QuerySteps[step] += 1
	q.QueryStep = step
}
func (q *QueryValidator) getLastBlockType() QueryStep {
	return q.QueryStep
}
func (q *QueryValidator) isValid() error {
	var err error
	if !q.GetQueryStep(INTERNAL_AS_TOKEN) {
		err = q.validateAllSelectStatmentUnaliasedFields()
		if err != nil {
			return err
		}
	} else {
		err = q.validateAllSelectStatmentAliasedFields()
		if err != nil {
			return err
		}
	}
	return nil
}
func (q *QueryValidator) validateAllSelectStatmentUnaliasedFields() error {
	for _, fieldStatement := range q.selectorFields {
		fields := FindUnaliasedFields(fieldStatement)
		err := q.validateTableFields("", fields...)
		if err != nil {
			return err
		}
	}
	return nil
}
func (q *QueryValidator) validateTableFields(tableAlias string, fields ...string) error {
	table := q.tableAliases[tableAlias]
	if table == nil {
		return ErrorDescription(ErrSyntax, fmt.Sprintf("Failed to resolve field alias [%s]. ", tableAlias))
	}

	for _, fieldName := range fields {
		_, exists := table.Fields[TableFieldName(fieldName)]
		if !exists {
			return ErrorDescription(ErrSyntax, fmt.Sprintf("%s does not exist in %s", fieldName, table.TableName))
		}
	}
	return nil
}
func (q *QueryValidator) validateAllSelectStatmentAliasedFields() error {
	for _, fieldStatement := range q.selectorFields {
		fields, found := RecoverSelectStatementAliasedFields(fieldStatement)
		if !found {
			return ErrorDescription(ErrSyntax, fmt.Sprintf("Found ambigous field in a select statement, perhaps a missing alias. Statement: %s", fieldStatement))
		}
		err := q.validateSelectStatementAliasedFields(fields, found)
		if err != nil {
			return err
		}
	}
	return nil
}

func (q *QueryValidator) validateSelectStatementAliasedFields(fields [][]string, found bool) error {
	var alias string
	var fieldName string

	for _, field := range fields {
		if found {
			alias = field[0]
			fieldName = field[1]
		} else {
			alias = ""
			fieldName = field[0]
		}

		if err := q.validateTableFields(alias, fieldName); err != nil {
			return err
		}
	}

	return nil
}

// Index refers to the index of the Alias-FieldName separator
func trimLeftIndex(str string, index int) string {
	for i := index; i >= 0; i-- {
		switch str[i] {
		case ':', ' ', '(':
			return str[i+1 : index]
		}
	}
	return str[:index]
}

// Index refers to the index of the Alias-FieldName separator
func trimRightIndex(str string, index int) string {
	for i := index; len(str) > i; i++ {
		switch str[i] {
		case ':', ' ', ')':
			return str[index+1 : i]
		}
	}
	return str[index+1:]
}

func RecoverSelectStatementAliasedField(str string, index int) (left, right string) {
	return trimLeftIndex(str, index), trimRightIndex(str, index)
}

func RecoverSelectStatementAliasedFields(str string) ([][]string, bool) {
	fields := [][]string{}
	for i, r := range str {
		if r == '.' {
			left, right := RecoverSelectStatementAliasedField(str, i)
			fields = append(fields, []string{left, right})
		}
	}
	return fields, len(fields) > 0
}

// Returns the first trimmed unalised field and the rest of the string
func BreakUnaliasedField(str string) (string, string) {
	left := -1
	right := -1
	found := false
outer:
	for i, r := range str {
		switch r {
		case '(':
			left = i
			// Update found to false since a inner has been found
			found = false
		case ')', ':', '+', '-', '=', '>', '<':
			// Does not mark the right end unless the word has been found
			if found {
				right = i
				break outer
			}
		case ' ':
			if found {
				right = i
				break outer
			} else {
				left = i
			}
		default:
			// Does not mark the word as found unless the left side has been found
			if left >= 0 {
				found = true
			}
		}
	}
	if right == -1 && left == -1 {
		return str, ""
	}
	if right == -1 || left == -1 {
		return str, ""
	}
	return str[left+1 : right], str[right:]
}
func FindUnaliasedFields(str string) []string {
	var fields []string
	var field string
	for {
		field, str = BreakUnaliasedField(str)
		if str == "" {
			break
		}
		fields = append(fields, field)
	}
	return fields
}

func (q *QueryValidator) registerForValidation(fieldNames ...string) {
	q.selectorFields = append(q.selectorFields, fieldNames...)
}
func newQueryValidator(t *TableRegistry) *QueryValidator {
	return &QueryValidator{
		QuerySteps: make(map[QueryStep]int),
		QueryStep:  -1,

		selectorFields: make([]string, 0),
		tableAliases:   make(map[string]*TableRegistry),

		TableRegistry: t,
	}
}

func newAdditionalSelectQuery(query *Query) *AdditionalSelectQuery {
	return &AdditionalSelectQuery{
		Query: query,
	}
}
func newPartialInnerJoinQuery(query *Query) *PartialInnerJoinQuery {
	return &PartialInnerJoinQuery{
		parentQuery: query,
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
		return q.SetError(table.Error.Error())
	}

	q.placeholderIndex = 1
	q.QueryValidator = newQueryValidator(t)

	return &q
}
