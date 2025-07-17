package configuration

const (
	DEBUGGING Environment = iota
	PRODUCTION
)

type Environment int

type EnvironmentSettings struct {
	Environment
}

var environment *EnvironmentSettings = &EnvironmentSettings{
	Environment: PRODUCTION,
}

func (e *EnvironmentSettings) SetEnvironment(en Environment) {
	e.Environment = en
}
func (e *EnvironmentSettings) GetEnvironment() Environment {
	return e.Environment
}
