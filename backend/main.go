package main

import (
	"fmt"
	"log"
	"net/http"

	blt "github.com/netivism/goshort/backend/pkg/bolt"
	"github.com/netivism/goshort/backend/pkg/restapi"
)

func main() {
	blt.New()
	router := restapi.New()
	port := ":33512"
	fmt.Printf("HTTP server startup and listening on port%s\n", port)
	log.Fatal(http.ListenAndServe(port, router))

	blt.Close()
}
