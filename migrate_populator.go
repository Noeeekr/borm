package borm

type MigrationPopulator struct {
	queries []*Query
}

// Register queries to be executed after the relation migration ends
func (m *MigrationPopulator) RegisterMigrationQueries(queries ...*Query) {
	m.queries = append(m.queries, queries...)
}

func (m *MigrationPopulator) GetMigrationQueries() <-chan *Query {
	queryIterator := make(chan *Query)
	go func() {
		for _, query := range m.queries {
			queryIterator <- query
		}
	}()
	return queryIterator
}

func newMigrationPopulator() *MigrationPopulator {
	return &MigrationPopulator{
		queries: []*Query{},
	}
}
