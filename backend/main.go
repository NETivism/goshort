package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	blt "github.com/netivism/goshort/backend/pkg/bolt"
	"github.com/netivism/goshort/backend/pkg/db"
	"github.com/netivism/goshort/backend/pkg/env"
	"github.com/netivism/goshort/backend/pkg/restapi"
)

func main() {
	err := env.Check()
	if err != nil {
		panic(err)
	}

	err = db.Connect()
	if err != nil {
		log.Fatal("Error when connect database")
	}

	if strings.ToLower(env.Get(env.DatabaseMigrate)) == "true" {
		_, err = os.Stat("goshort.db")
		if err == nil {
			blt.Migrate()
		}
		os.Exit(0)
	}
	router := restapi.New()
	port := ":" + env.Get(env.ListenPort)
	fmt.Printf("HTTP server startup and listening on port%s\n", port)
	log.Fatal(http.ListenAndServe(port, router))

}
