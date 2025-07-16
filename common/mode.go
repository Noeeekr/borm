package common

import "os"

type ApplicationEnvironment int

const (
	DEBUGGING ApplicationEnvironment = iota
	PRODUCTION
)

var mode = PRODUCTION

func Environment() ApplicationEnvironment {
	return mode
}

func init() {
	for _, arg := range os.Args {
		if arg == "--debug" {
			mode = DEBUGGING
		}
	}
}
