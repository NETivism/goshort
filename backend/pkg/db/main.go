package db

import (
	"encoding/json"
	"log"
	"net/url"
	"os"
	"sync"

	//	"gorm.io/driver/mysql"
	//	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	blt "github.com/netivism/goshort/backend/pkg/bolt"
	"github.com/netivism/goshort/backend/pkg/env"
	"github.com/netivism/goshort/backend/pkg/goshort"
	"github.com/netivism/goshort/backend/pkg/model"
	bolt "go.etcd.io/bbolt"
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
		case "sqlite":
			dialector = sqlite.Open(env.GetSqliteDsn())
		/*
			case "mysql":
				dialector = mysql.Open(env.GetMysqlDsn())
			case "postgres":
				dialector = postgres.Open(env.GetPostgresDsn())
		*/
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

func Migrate() {
	if _, err := os.Stat("goshort.db"); err == nil {
		blt.New()
		dbi, err := Connect()
		if err != nil {
			log.Fatal("Migration stopped because destination database connection failed.")
		}
		log.Printf("Start database migration")

		blt.DB.View(func(tx *bolt.Tx) error {
			bucket := tx.Bucket(blt.GoshortBucket)
			c := bucket.Cursor()
			for k, v := c.First(); k != nil; k, v = c.Next() {
				var val goshort.GoShort
				err := json.Unmarshal(v, &val)
				if err == nil {
					r, _ := url.Parse(val.Redirect)
					record := model.Redirect{
						Id:       val.Short,
						Redirect: val.Redirect,
						Domain:   r.Host,
						Path:     r.Path,
					}
					dbi.Create(&record)
				}
			}
			return nil
		})
		log.Printf("Database migration complete")
		blt.Close()
	} else {
		log.Fatal("Could not find old boltdb goshort.db")
	}

}
