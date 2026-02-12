package middleware

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/netivism/goshort/backend/pkg/env"
	"github.com/netivism/goshort/backend/pkg/handler"
)

// Auth dispatches to the appropriate auth middleware based on AUTH_TYPE.
func Auth(next http.HandlerFunc) http.HandlerFunc {
	authType := env.Get(env.AuthType)
	if authType == "apikey" {
		return ApiKeyAuth(next)
	}
	return BasicAuth(next)
}

// ApiKeyAuth validates an Authorization: Bearer <key> header.
func ApiKeyAuth(next http.HandlerFunc) http.HandlerFunc {
	apiKey := env.Get(env.AuthApikey)

	if apiKey == "" {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handler.HandlerError(w, "Page not available.", http.StatusForbidden)
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		providedHash := sha256.Sum256([]byte(parts[1]))
		expectedHash := sha256.Sum256([]byte(apiKey))

		if subtle.ConstantTimeCompare(providedHash[:], expectedHash[:]) != 1 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
