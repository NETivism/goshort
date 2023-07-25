package restapi

import (
	"github.com/gorilla/mux"
	"github.com/netivism/goshort/backend/pkg/restapi/handle"
	"github.com/netivism/goshort/backend/pkg/restapi/list"
	"github.com/netivism/goshort/backend/pkg/restapi/middleware"
	"github.com/netivism/goshort/backend/pkg/restapi/root"
)

// Root route handler. will get all path by id
// HandlerError is fallback function when entry not found

func New() *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/list/entries", middleware.BasicAuth(list.Entries)).Methods("GET")
	// router.HandleFunc("/handle/list-entry/{id}", BasicAuth(list.Entry)).Methods("GET")
	router.HandleFunc("/handle/create", handle.Create).Methods("POST")
	// router.HandleFunc("/handle/update-entry/{id}", BasicAuth(handle.Update)).Methods("PUT")
	router.HandleFunc("/{id}", root.Root).Methods("GET")

	return router
}
