package main

import (
	"fmt"
	"log"
	"net/http"

	// blt "github.com/netivism/goshort/backend/pkg/bolt"
	"github.com/netivism/goshort/backend/pkg/db"
	"github.com/netivism/goshort/backend/pkg/env"
	"github.com/netivism/goshort/backend/pkg/restapi"
)

func main() {
	err := env.Check()
	if err != nil {
		panic(err)
	}

	db.Connect()
	// blt.New()
	router := restapi.New()
	port := ":" + env.Get(env.ListenPort)
	fmt.Printf("HTTP server startup and listening on port%s\n", port)
	log.Fatal(http.ListenAndServe(port, router))

	// blt.Close()
}
