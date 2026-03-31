package root

import (
	"log"
	"net/http"
	"regexp"

	"github.com/gorilla/mux"
	"github.com/netivism/goshort/backend/pkg/db"
	"github.com/netivism/goshort/backend/pkg/handler"
	"github.com/netivism/goshort/backend/pkg/model"
	"github.com/netivism/goshort/backend/pkg/referrer"
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

	dbi := db.Get()
	exists := model.Redirect{Id: shortenId}
	result := dbi.Limit(1).Find(&exists)

	if result.RowsAffected > 0 && exists.Redirect != "" {
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "Thu, 01 Jan 1970 00:00:00 GMT")
		http.Redirect(w, req, exists.Redirect, http.StatusMovedPermanently)

		scheme := "http"
		if req.TLS != nil {
			scheme = "https"
		}
		if fwd := req.Header.Get("X-Forwarded-Proto"); fwd != "" {
			scheme = fwd
		}
		currentURL := scheme + "://" + req.Host + req.RequestURI
		refererHeader := req.Header.Get("Referer")
		userAgent := req.Header.Get("User-Agent")

		info := referrer.Parse(currentURL, refererHeader, userAgent)

		visit := model.Visits{
			RedirectId: shortenId,
			Referer: model.Referer{
				Type:    info.Type,
				Network: info.Network,
				Link:    info.Link,
			},
		}
		result = dbi.Create(&visit)
		if result.Error != nil {
			log.Printf("%s\n", result.Error)
		}
	}
}
