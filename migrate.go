package borm

import (
	"fmt"
	"strings"

	"github.com/Noeeekr/borm/common"
	"github.com/Noeeekr/borm/internal/registers"
)

type Migrate struct {
	*MigrateEnvironment
	*MigrateRelations
}
type MigrateRelations struct{}
type MigrateEnvironment struct{}

func NewMigrate() *Migrate {
	return &Migrate{
		MigrateEnvironment: &MigrateEnvironment{},
		MigrateRelations:   &MigrateRelations{},
	}
}

func NewRelationMigrate() *MigrateRelations {
	return &MigrateRelations{}
}

func NewEnvironmentMigrate() *MigrateEnvironment {
	return &MigrateEnvironment{}
}

func (m *MigrateEnvironment) Environment() *common.Error {
	return nil
}
func (m *MigrateRelations) Relations() *common.Error {
	// Create the relation queries
	queries := []string{}
	for _, table := range *registers.Tables {
		if table.Error != nil {
			return table.Error
		}
		queries = append(queries, parseCreateQuery(table))
	}

	return nil
}
func parseCreateQuery(table *registers.Table) string {
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

	return fmt.Sprintf("CREATE TABLE %s (%s\n);", table.Name, strings.Join(fields, ","))
}
