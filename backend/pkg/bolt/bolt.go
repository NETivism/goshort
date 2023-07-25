package bolt

import (
	"log"

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

func Close() {
	defer DB.Close()
}
