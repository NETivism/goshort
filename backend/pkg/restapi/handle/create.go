package handle

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	blt "github.com/netivism/goshort/backend/pkg/bolt"
	"github.com/netivism/goshort/backend/pkg/goshort"
	"github.com/netivism/goshort/backend/pkg/handler"
	"github.com/pkg/errors"
	"github.com/speps/go-hashids"
	bolt "go.etcd.io/bbolt"
)

func Create(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	var gs goshort.GoShort
	err := json.NewDecoder(req.Body).Decode(&gs)
	if err != nil {
		handler.HandlerError(w, "Could not decode entry", http.StatusBadRequest)
		return
	}

	if gs.Redirect == "" {
		handler.HandlerError(w, "Entry missing redirect address", http.StatusBadRequest)
		return
	}

	uhash, err := GenerateUniqueHash()
	if err != nil {
		handler.HandlerError(w, fmt.Sprintf("Hash generation error. %v", err), http.StatusBadRequest)
		return
	}

	err = blt.DB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blt.GoshortBucket)
		gs.Short = uhash
		gs.Count = 1
		jurl, err := json.Marshal(gs)
		if err != nil {
			return errors.Wrap(err, "Err: could not marshal entry")
		}
		err = bucket.Put([]byte(uhash), jurl)
		log.Println("SUCCESS: New entry created - " + string(jurl))
		if err != nil {
			return errors.Wrap(err, "Err: could not put data into bucket")
		}
		return nil
	})

	if err != nil {
		handler.HandlerError(w, fmt.Sprintf("Error when saving record. %v", err), http.StatusInternalServerError)
	} else {
		result := []goshort.GoShort{gs}
		handler.HandlerSuccess(w, "URL shorten successfully.", result, http.StatusCreated)
	}
}

func GenerateUniqueHash() (string, error) {
	tries := 0
	now := time.Now().Unix()

	for tries < 5 {
		hd := hashids.NewData()
		hd.MinLength = 5
		hd.Salt = fmt.Sprintf("%d", rand.Intn(10000000))
		h, _ := hashids.NewWithData(hd)
		uhash, _ := h.Encode([]int{int(now)})

		err := blt.DB.View(func(tx *bolt.Tx) error {
			bucket := tx.Bucket(blt.GoshortBucket)
			raw := bucket.Get([]byte(uhash))
			if raw == nil {
				return nil
			}
			time.Sleep(1 * time.Second)
			err := errors.New("Duplicate entry of " + uhash + ", continue")
			return err
		})

		if err == nil {
			return uhash, nil
		} else {
			log.Printf("Continue next loop because %s\n", err)
		}

		tries++
	}

	return "", errors.New("Unable to generate unique hash")
}
