package manager

type Configuration struct {
	ignoreExisting    bool
	reacreateExisting bool
	undoOnError       bool
}

func NewConfiguration() *Configuration {
	return &Configuration{}
}

func (c *Configuration) RecreateExisting() *Configuration {
	c.reacreateExisting = true
	return c
}
func (c *Configuration) IgnoreExisting() *Configuration {
	c.ignoreExisting = true
	return c
}
func (c *Configuration) UndoOnError() *Configuration {
	c.undoOnError = true
	return c
}
