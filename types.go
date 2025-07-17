package borm

import (
	"github.com/Noeeekr/borm/configuration"
	"github.com/Noeeekr/borm/internal/registers"
)

const (
	INSERT = registers.INSERT
	DELETE = registers.DELETE
	UPDATE = registers.UPDATE
	SELECT = registers.SELECT

	ALL = registers.ALL
)

const (
	DEBUGGING  = configuration.DEBUGGING
	PRODUCTION = configuration.PRODUCTION
)
