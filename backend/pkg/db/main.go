package db

import (
	"log"
	"sync"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/netivism/goshort/backend/pkg/env"
	"github.com/netivism/goshort/backend/pkg/model"
)

var (
	instance *gorm.DB
	once     sync.Once
)

func Connect() (*gorm.DB, error) {
	var err error
	once.Do(func() {
		dbType := env.Get(env.DatabaseType)
		var dialector gorm.Dialector

		switch dbType {
		case "mysql":
			dialector = mysql.Open(env.GetMysqlDsn())
		case "sqlite":
			dialector = sqlite.Open(env.GetSqliteDsn())
		case "postgres":
			dialector = postgres.Open(env.GetPostgresDsn())
		default:
			log.Fatalf("Unsupported DATABASE_TYPE: %s", dbType)
		}

		instance, err = gorm.Open(dialector, &gorm.Config{})
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		} else {
			// auto create table when not exists
			instance.AutoMigrate(&model.Redirect{}, &model.Visits{})
		}
	})

	return instance, err
}
