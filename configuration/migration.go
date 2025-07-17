package configuration

type MigrationSettings struct {
	Enabled  bool
	Ignore   bool
	Recreate bool
	Undo     bool
}

var migration *MigrationSettings = &MigrationSettings{}

func (m *MigrationSettings) Enable() *MigrationSettings {
	m.Enabled = true
	return m
}
func (m *MigrationSettings) RecreateExisting() *MigrationSettings {
	m.Recreate = true
	return m
}
func (m *MigrationSettings) IgnoreExisting() *MigrationSettings {
	m.Ignore = true
	return m
}
func (m *MigrationSettings) UndoOnError() *MigrationSettings {
	m.Undo = true
	return m
}
