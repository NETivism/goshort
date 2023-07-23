package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"crypto/sha256"
	"crypto/subtle"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/speps/go-hashids/v2"
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

type GoShortResp struct {
	Success int       `json:"success"`
	Message string    `json:"message"`
	Result  []GoShort `json:"result,omitempty"`
}

type Authenticate struct {
	username string
	password string
}

func BasicAuth(next http.HandlerFunc) http.HandlerFunc {
	auth := new(Authenticate)
	auth.username = os.Getenv("AUTH_USERNAME")
	auth.password = os.Getenv("AUTH_PASSWORD")

	if auth.username == "" {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			HandlerError(w, "Page not available.", http.StatusForbidden)
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok {
			if username == "" {
				HandlerError(w, "No user name provided", http.StatusUnauthorized)
				return
			}
			usernameHash := sha256.Sum256([]byte(username))
			passwordHash := sha256.Sum256([]byte(password))
			expectedUsernameHash := sha256.Sum256([]byte(auth.username))
			expectedPasswordHash := sha256.Sum256([]byte(auth.password))

			usernameMatch := (subtle.ConstantTimeCompare(usernameHash[:], expectedUsernameHash[:]) == 1)
			passwordMatch := (subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1)

			if usernameMatch && passwordMatch {
				next.ServeHTTP(w, r)
				return
			}
		}

		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

func ListEntries(w http.ResponseWriter, req *http.Request) {
	queryParams := req.URL.Query()
	offsetStr := queryParams.Get("offset")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}
	limit := 100

	result := make([]GoShort, 0)

	db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(goshortBucket)
		c := bucket.Cursor()

		count := -1
		for k, v := c.First(); k != nil; k, v = c.Next() {
			count++
			if count < offset {
				continue
			}
			var val GoShort
			err := json.Unmarshal(v, &val)

			if err == nil {
				result = append(result, val)
				if len(result) > limit {
					break
				}
			}
		}

		return nil
	})
	if len(result) > 0 {
		HandlerSuccess(w, "Entries loaded successfully.", result, http.StatusOK)
	} else {
		HandlerSuccess(w, "No entry found.", result, http.StatusOK)
	}
}

// CreateEntry route to create shorten entry
func CreateEntry(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	var goshort GoShort
	err := json.NewDecoder(req.Body).Decode(&goshort)
	if err != nil {
		HandlerError(w, "Could not decode entry", http.StatusBadRequest)
		return
	}

	if goshort.Redirect == "" {
		HandlerError(w, "Entry missing redirect address", http.StatusBadRequest)
		return
	}

	uhash, err := GenerateUniqueHash()
	if err != nil {
		HandlerError(w, fmt.Sprintf("Hash generation error. %v", err), http.StatusBadRequest)
		return
	}

	err = db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(goshortBucket)
		goshort.Short = uhash
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
		HandlerError(w, fmt.Sprintf("Error when saving record. %v", err), http.StatusInternalServerError)
	} else {
		result := []GoShort{goshort}
		HandlerSuccess(w, "URL shorten successfully.", result, http.StatusCreated)
	}
}

// GenerateUniqueHash generates a unique hash using hashids library.
// It returns the generated hash as a string and any error encountered during the process.
func GenerateUniqueHash() (string, error) {
	tries := 0
	now := time.Now().Unix()

	for tries < 5 {
		hd := hashids.NewData()
		hd.MinLength = 5
		hd.Salt = fmt.Sprintf("%d", rand.Intn(10000000))
		h, _ := hashids.NewWithData(hd)
		uhash, _ := h.Encode([]int{int(now)})

		err := db.View(func(tx *bolt.Tx) error {
			bucket := tx.Bucket(goshortBucket)
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

// Root route handler. will get all path by id
func Redirect(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	shortenId := params["id"]
	if shortenId == "" {
		HandlerError(w, "Invalid ID", http.StatusNotFound)
		return
	}

	if len(shortenId) > 20 {
		HandlerError(w, "Invalid ID", http.StatusNotFound)
		return
	}

	regex := regexp.MustCompile("^[a-zA-Z0-9]+$")
	match := regex.MatchString(shortenId)
	if !match {
		HandlerError(w, "Invalid ID", http.StatusNotFound)
		return
	}

	var goshort GoShort
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(goshortBucket)
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
		HandlerError(w, fmt.Sprintf("URL not found. %s", err), http.StatusNotFound)
	} else {
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
		http.Redirect(w, req, goshort.Redirect, http.StatusMovedPermanently)
	}
}

// HandlerError is fallback function when entry not found
func HandlerError(w http.ResponseWriter, message string, status int) {
	err := errors.New("Err: " + message)
	log.Printf("%s\n", err)
	result := make([]GoShort, 0)

	goShortResp := GoShortResp{
		Success: 0,
		Message: message,
		Result:  result,
	}
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(goShortResp)
}

func HandlerSuccess(w http.ResponseWriter, message string, result []GoShort, status int) {
	goShortResp := GoShortResp{
		Success: 1,
		Message: message,
		Result:  result,
	}
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(goShortResp)
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
	router.HandleFunc("/handle/list-entries", BasicAuth(ListEntries)).Methods("GET")
	// router.HandleFunc("/handle/list-entry/{id}", BasicAuth(ListEntries)).Methods("GET")
	// router.HandleFunc("/handle/update-entry/{id}", BasicAuth(ListEntries)).Methods("PUT")
	router.HandleFunc("/handle/create-entry", CreateEntry).Methods("POST")
	router.HandleFunc("/{id}", Redirect).Methods("GET")
	log.Fatal(http.ListenAndServe(":33512", router))
	log.Println("Server started at port 33512")

	defer db.Close()
}
