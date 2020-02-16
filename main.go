package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
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

type GoShort struct {
	Short    string `json:"short"`
	Redirect string `json:"redirect"`
	Count    int    `json:count`
}

type securedFileSystem struct {
	fs http.FileSystem
}

func (sfs securedFileSystem) Open(path string) (http.File, error) {
	f, err := sfs.fs.Open(path)
	if err != nil {
		return nil, err
	}

	s, err := f.Stat()
	if s.IsDir() {
		index := strings.TrimSuffix(path, "/") + "/index.html"
		if _, err := sfs.fs.Open(index); err != nil {
			return nil, err
		}
	}

	return f, nil
}

func CreateEndpoint(w http.ResponseWriter, req *http.Request) {
	var goshort GoShort
	err := json.NewDecoder(req.Body).Decode(&goshort)
	if err != nil {
		notFound(w)
		return
	}
	if goshort.Redirect != "" {
		uhash := UrlHash(goshort.Redirect)
		goshort.Short = uhash

		err = db.Update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket(goshortBucket)
			goshort.Count = 1
			jurl, err := json.Marshal(goshort)
			if err != nil {
				return errors.Wrap(err, "could not marshal entry")
			}
			err = bucket.Put([]byte(uhash), jurl)
			if err != nil {
				return errors.Wrap(err, "could not put data into bucket")
			}
			return nil
		})
		if err != nil {
			notFound(w)
			return
		} else {
			json.NewEncoder(w).Encode(goshort)
			return
		}
	}
	notFound(w)
}

func RootEndpoint(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	log.Output(1, params["id"])
	var goshort GoShort
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(goshortBucket)
		raw := bucket.Get([]byte(params["id"]))
		fmt.Printf("The answer is: %s\n", raw)
		if raw == nil {
			return errors.New("No entry found")
		} else {
			err := json.Unmarshal(raw, &goshort)
			if err != nil {
				return errors.Wrap(err, "could not unmarshal entry")
			} else {
				return nil
			}
		}
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
		http.Redirect(w, req, goshort.Redirect, 301)
		return
	}
	notFound(w)
}

func notFound(w http.ResponseWriter) {
	w.WriteHeader(404)
	fmt.Fprint(w, "404 not found")
}

func UrlHash(s string) string {
	hd := hashids.NewData()
	h, _ := hashids.NewWithData(hd)
	now := time.Now()
	hid, _ := h.Encode([]int{int(now.Unix())})
	return hid
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

	fs := http.FileServer(securedFileSystem{http.Dir("./public")})
	router := mux.NewRouter()
	router.HandleFunc("/create", CreateEndpoint).Methods("PUT")
	router.HandleFunc("/{id}", RootEndpoint).Methods("GET")
	router.PathPrefix("/").Handler(http.StripPrefix("/public/", fs))
	log.Fatal(http.ListenAndServe(":33512", router))

	defer db.Close()
}
