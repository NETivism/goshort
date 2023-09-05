package main

import (
	"fmt"
	"log"
	"net/http"

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

	// doing migration from bolddb to sqlite
	if env.Get(env.DatabaseMigrate) != "" {
		db.Migrate()
	} else {
		router := restapi.New()
		port := ":" + env.Get(env.ListenPort)
		fmt.Printf("HTTP server startup and listening on port%s\n", port)
		log.Fatal(http.ListenAndServe(port, router))
	}
}
