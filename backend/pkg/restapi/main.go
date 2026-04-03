package restapi

import (
	"github.com/gorilla/mux"
	"github.com/netivism/goshort/backend/pkg/restapi/handle"
	"github.com/netivism/goshort/backend/pkg/restapi/middleware"
	"github.com/netivism/goshort/backend/pkg/restapi/root"
)

// Root route handler. will get all path by id
// HandlerError is fallback function when entry not found

func New() *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/handle/create", handle.Create).Methods("POST")
	router.HandleFunc("/handle/create-entry", handle.Create).Methods("POST") // backward compatibility
	router.HandleFunc("/handle/visits/{id}", middleware.Auth(handle.Visits)).Methods("GET")
	router.HandleFunc("/{id}", root.Root).Methods("GET")

	return router
}
