package bolt

import (
	"encoding/json"
	"log"
	"net/url"
	"strings"

	"github.com/netivism/goshort/backend/pkg/db"
	"github.com/netivism/goshort/backend/pkg/goshort"
	"github.com/netivism/goshort/backend/pkg/model"
	bolt "go.etcd.io/bbolt"
)

var (
	DB            *bolt.DB
	GoshortBucket = []byte("goshort")
)

func New() {
	var err error
	DB, err = bolt.Open("goshort.db", 0666, nil)
	if err != nil {
		log.Fatal(err)
	}

	err = DB.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(GoshortBucket); err != nil {
			log.Fatal(err)
		}
		return err
	})
}

func Migrate() {
	log.Println("Migration Start")
	New()
	items := make([]goshort.GoShort, 0)
	DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(GoshortBucket)
		c := bucket.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			var val goshort.GoShort
			err := json.Unmarshal(v, &val)
			if err == nil {
				items = append(items, val)
			}
		}
		return nil
	})
	Close()
	dbi := db.Get()
	count := 0
	redirects := []model.Redirect{}
	for _, gs := range items {
		count++
		gs.Redirect = strings.Replace(gs.Redirect, "\n", "", -1)
		r, err := url.Parse(gs.Redirect)
		if err == nil {
			redirect := model.Redirect{
				Id:       gs.Short,
				Redirect: gs.Redirect,
				Domain:   r.Host,
				Path:     r.Path,
			}
			redirects = append(redirects, redirect)
		} else {
			log.Printf("Error in parsing url %v of shorten id %v", gs.Redirect, gs.Short)
		}
	}
	log.Printf("Trying to insert %v records..", count)
	dbi.CreateInBatches(&redirects, 1000)
	log.Println("Migration Completed. Exit application.")
}

func Close() {
	defer DB.Close()
}
