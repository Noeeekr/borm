package manager

import (
	"fmt"
	"strings"

	"github.com/Noeeekr/borm/common"
	"github.com/Noeeekr/borm/internal/registers"
)

func (m *DatabaseManager) Relations() *common.Error {
	// Create the relation queries
	queries := []*registers.Query{}

	for _, table := range *m.TableCache {
		if table.Error != nil {
			return table.Error
		}
		queries = append(queries, parseCreateTableQuery(table))
	}

	for _, query := range queries {
		fmt.Println(query)
	}

	return nil
}

func parseCreateTableQuery(table *registers.Table) *registers.Query {
	var fields []string
	// Parse fields to query
	for _, field := range table.Fields {
		query := fmt.Sprintf("\n\t%s %s", field.Name, field.Type)
		if field.Constraints != "" {
			query += fmt.Sprintf(" %s", field.Constraints)
		}
		if field.ForeignKey != "" {
			query += fmt.Sprintf(",%s", field.ForeignKey)
		}
		fields = append(fields, query)
	}

	q := registers.Query{}
	q.Query = fmt.Sprintf("CREATE TABLE %s (%s\n);", table.Name, strings.Join(fields, ","))
	return &q
}
