package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/speps/go-hashids"
	bolt "go.etcd.io/bbolt"
)

var (
	goshortBucket = []byte("goshort")
	db            *bolt.DB
)

// GoShort is the object to save to bolt
type GoShort struct {
	Short    string `json:"short"`
	Redirect string `json:"redirect"`
	Count    int    `json:"count"`
}

// CreateEntry route to create shorten entry
func CreateEntry(w http.ResponseWriter, req *http.Request) {
	var goshort GoShort
	err := json.NewDecoder(req.Body).Decode(&goshort)
	if err != nil {
		NotFound(w)
		err = errors.New("Err: could not decode entry")
		log.Printf("%s\n", err)
		return
	}
	if goshort.Redirect != "" {
		hd := hashids.NewData()
		limit := 0
		now := time.Now().Unix()
		uhash := ""
		for limit < 5 {
			hd.Salt = fmt.Sprintf("%d", rand.Intn(10000000))
			h, _ := hashids.NewWithData(hd)
			uhash, _ = h.Encode([]int{int(now)})
			err := db.View(func(tx *bolt.Tx) error {
				bucket := tx.Bucket(goshortBucket)
				raw := bucket.Get([]byte(uhash))
				if raw == nil {
					return nil
				}
				time.Sleep(1 * time.Second)
				err := errors.New("Err: duplicate entry or " + uhash + ", continue")
				return err
			})
			if err == nil {
				break
			} else {
				log.Printf("Continue next loop because %s\n", err)
			}
			limit++
		}

		err = db.Update(func(tx *bolt.Tx) error {
			if len(uhash) <= 0 {
				return errors.Wrap(err, "No short string")
			} else {
				goshort.Short = uhash
			}
			bucket := tx.Bucket(goshortBucket)
			goshort.Count = 1
			jurl, err := json.Marshal(goshort)
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
			log.Printf("%s\n", err)
			NotFound(w)
		} else {
			json.NewEncoder(w).Encode(goshort)
		}
	} else {
		NotFound(w)
		err = errors.New("Err: entry missing redirect address")
		log.Printf("%s\n", err)
	}
}

// Root route handler. will get all path by id
func Root(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	var goshort GoShort
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(goshortBucket)
		raw := bucket.Get([]byte(params["id"]))
		if raw == nil {
			return errors.New("No entry found")
		}
		log.Printf("SUCCESS: visit %s\n", raw)
		err := json.Unmarshal(raw, &goshort)
		if err != nil {
			return errors.Wrap(err, "could not unmarshal entry")
		}
		return nil
	})

	if err == nil {
		err = db.Update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket(goshortBucket)
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
		http.Redirect(w, req, goshort.Redirect, 301)
	} else {
		NotFound(w)
		log.Printf("%s\n", err)
	}
}

// NotFound is fallback function when entry not found
func NotFound(w http.ResponseWriter) {
	w.WriteHeader(404)
	fmt.Fprint(w, "404 not found")
}

func main() {
	var err error
	db, err = bolt.Open("goshort.db", 0666, nil)
	err = db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(goshortBucket); err != nil {
			log.Fatal(err)
		}
		return err
	})

	router := mux.NewRouter()
	router.HandleFunc("/handle/create-entry", CreateEntry).Methods("PUT")
	router.HandleFunc("/{id}", Root).Methods("GET")
	log.Fatal(http.ListenAndServe(":33512", router))

	defer db.Close()
}
