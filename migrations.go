package borm

import (
	"fmt"
	"reflect"
	"strings"
)

var create_queries []string
var tables_to_drop []string

func GetCreateQueries() []string {
	return create_queries
}
func PrepareToDrop(t ...any) *Error {
	typ := reflect.TypeOf(t)
	if res := isStruct(typ); res != nil {
		return res
	}
	tableName := typ.Name()

	tables_to_drop = append(tables_to_drop, tableName)

	return nil
}
func PrepareToCreate(tables ...any) *Error {
	for _, t := range tables {
		err := prepareToCreateOne(t)
		if err != nil {
			return err
		}
	}
	return nil
}

func prepareToCreateOne(t any) *Error {
	// Reflect table information
	tableInformation, response := tables.Table(t)
	if response != nil {
		return response
	}

	var fields []string
	// Parse fields to query
	for _, field := range tableInformation.Fields {
		query := fmt.Sprintf("\n\t%s %s", field.Name, field.Type)
		if field.Constraints != "" {
			query += fmt.Sprintf(" %s", field.Constraints)
		}
		if field.ForeignKey != "" {
			query += fmt.Sprintf(",%s", field.ForeignKey)
		}
		fields = append(fields, query)
	}
	query := fmt.Sprintf("CREATE TABLE %s (%s\n);", tableInformation.Name, strings.Join(fields, ","))

	create_queries = append(create_queries, query)

	return nil
}
