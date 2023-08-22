package root

import (
	"log"
	"net/http"
	"regexp"

	"github.com/gorilla/mux"
	"github.com/netivism/goshort/backend/pkg/db"
	"github.com/netivism/goshort/backend/pkg/handler"
	"github.com/netivism/goshort/backend/pkg/model"
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

	dbi, err := db.Connect()
	if err != nil {
		return
	}
	exists := model.Redirect{Id: shortenId}
	result := dbi.First(&exists)

	if result.RowsAffected > 0 && exists.Redirect != "" {
		http.Redirect(w, req, exists.Redirect, http.StatusMovedPermanently)
		visit := model.Visits{
			RedirectId: shortenId,
		}
		result = dbi.Create(&visit)
		if result.Error != nil {
			log.Printf("%s\n", result.Error)
		}
	}
}
