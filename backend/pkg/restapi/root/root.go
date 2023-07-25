package root

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"

	"github.com/gorilla/mux"
	blt "github.com/netivism/goshort/backend/pkg/bolt"
	"github.com/netivism/goshort/backend/pkg/goshort"
	"github.com/netivism/goshort/backend/pkg/handler"
	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
)

func Root(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	shortenId := params["id"]
	if shortenId == "" {
		handler.HandlerError(w, "Invalid ID", http.StatusNotFound)
		return
	}

	if len(shortenId) > 20 {
		handler.HandlerError(w, "Invalid ID", http.StatusNotFound)
		return
	}

	regex := regexp.MustCompile("^[a-zA-Z0-9]+$")
	match := regex.MatchString(shortenId)
	if !match {
		handler.HandlerError(w, "Invalid ID", http.StatusNotFound)
		return
	}

	var goshort goshort.GoShort
	err := blt.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blt.GoshortBucket)
		raw := bucket.Get([]byte(shortenId))
		if raw == nil {
			return errors.New("No entry of " + shortenId)
		}
		err := json.Unmarshal(raw, &goshort)
		if err != nil {
			return errors.Wrap(err, "Entry format error.")
		}
		return nil
	})

	if err != nil {
		handler.HandlerError(w, fmt.Sprintf("URL not found. %s", err), http.StatusNotFound)
	} else {
		err = blt.DB.Update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket(blt.GoshortBucket)
			goshort.Count++
			jurl, err := json.Marshal(goshort)
			if err != nil {
				return errors.Wrap(err, "could not marshal entry")
			}
			err = bucket.Put([]byte(goshort.Short), jurl)
			if err != nil {
				return errors.Wrap(err, "could not put data into bucket")
			}
			return nil
		})
		if err != nil {
			log.Printf("%s\n", err)
		}
		http.Redirect(w, req, goshort.Redirect, http.StatusMovedPermanently)
	}
}
