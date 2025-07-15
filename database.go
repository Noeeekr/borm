package borm

import (
	_ "github.com/lib/pq"

	"github.com/Noeeekr/borm/common"
	"github.com/Noeeekr/borm/internal/manager"
)

func On(user, password, host, db string) (*manager.DatabaseManager, *common.Error) {
	return manager.Connect(user, password, host, db)
}

func NewConfiguration() *manager.Configuration {
	return manager.NewConfiguration()
}
