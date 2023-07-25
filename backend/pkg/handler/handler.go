package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/netivism/goshort/backend/pkg/goshort"
	"github.com/pkg/errors"
)

func HandlerError(w http.ResponseWriter, message string, status int) {
	err := errors.New("Err: " + message)
	log.Printf("%s\n", err)
	result := make([]goshort.GoShort, 0)

	goShortResp := goshort.GoShortResp{
		Success: 0,
		Message: message,
		Result:  result,
	}
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(goShortResp)
}

func HandlerSuccess(w http.ResponseWriter, message string, result []goshort.GoShort, status int) {
	goShortResp := goshort.GoShortResp{
		Success: 1,
		Message: message,
		Result:  result,
	}
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(goShortResp)
}
