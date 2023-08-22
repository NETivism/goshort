package middleware

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"

	"github.com/netivism/goshort/backend/pkg/env"
	"github.com/netivism/goshort/backend/pkg/handler"
)

type Authenticate struct {
	username string
	password string
}

func BasicAuth(next http.HandlerFunc) http.HandlerFunc {
	auth := new(Authenticate)
	auth.username = env.Get(env.AuthUsername)
	auth.password = env.Get(env.AuthPassword)

	if auth.username == "" {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handler.HandlerError(w, "Page not available.", http.StatusForbidden)
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok {
			if username == "" {
				handler.HandlerError(w, "No user name provided", http.StatusUnauthorized)
				return
			}
			usernameHash := sha256.Sum256([]byte(username))
			passwordHash := sha256.Sum256([]byte(password))
			expectedUsernameHash := sha256.Sum256([]byte(auth.username))
			expectedPasswordHash := sha256.Sum256([]byte(auth.password))

			usernameMatch := (subtle.ConstantTimeCompare(usernameHash[:], expectedUsernameHash[:]) == 1)
			passwordMatch := (subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1)

			if usernameMatch && passwordMatch {
				next.ServeHTTP(w, r)
				return
			}
		}

		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}
