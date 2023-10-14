package handle

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"github.com/netivism/goshort/backend/pkg/db"
	"github.com/netivism/goshort/backend/pkg/model"

	"github.com/netivism/goshort/backend/pkg/goshort"
	"github.com/netivism/goshort/backend/pkg/handler"
	"github.com/pkg/errors"
	"github.com/speps/go-hashids"
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

	r, err := url.Parse(gs.Redirect)
	if err != nil {
		handler.HandlerError(w, "Redirection format must be http/s url.", http.StatusBadRequest)
		return
	}
	if r.Scheme != "http" && r.Scheme != "https" {
		handler.HandlerError(w, "Redirection format must be http/s url.", http.StatusBadRequest)
		return
	}
	if r.User.Username() != "" {
		handler.HandlerError(w, "Redirection format shouldn't include username of password.", http.StatusBadRequest)
		return
	}

	uhash, err := GenerateUniqueHash()
	if err != nil {
		handler.HandlerError(w, fmt.Sprintf("Hash generation error. %v", err), http.StatusBadRequest)
		return
	}

	dbi := db.Get()
	gs.Short = uhash
	gs.Count = 1
	record := model.Redirect{
		Id:       gs.Short,
		Redirect: gs.Redirect,
		Domain:   r.Host,
		Path:     r.Path,
	}
	created := dbi.Create(&record)

	if created.Error != nil {
		handler.HandlerError(w, fmt.Sprintf("Error when saving record. %v", err), http.StatusInternalServerError)
	} else {
		result := []goshort.GoShort{gs}
		handler.HandlerSuccess(w, "URL shorten successfully.", result, http.StatusCreated)
	}
}

func GenerateUniqueHash() (string, error) {
	tries := 0
	now := time.Now().Unix()
	dbi := db.Get()

	for tries < 5 {
		hd := hashids.NewData()
		hd.MinLength = 5
		hd.Salt = fmt.Sprintf("%d", rand.Intn(10000000))
		h, _ := hashids.NewWithData(hd)
		uhash, _ := h.Encode([]int{int(now)})

		var exists = model.Redirect{Id: uhash}
		result := dbi.Limit(1).Find(&exists)
		if result.RowsAffected > 0 {
			log.Printf("Continue next loop because conflict of short id %s", uhash)
			tries++
			continue
		}
		return uhash, nil
	}

	return "", errors.New("Unable to generate unique hash")
}
