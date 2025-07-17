package configuration

type Configuration struct{}

var configuration *Configuration = &Configuration{}

func Settings() *Configuration {
	return configuration
}

func (c *Configuration) Environment() *EnvironmentSettings {
	return environment
}
func (c *Configuration) Migrations() *MigrationSettings {
	return migration
}
