package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func okHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func TestBasicAuthNoAuth(t *testing.T) {
	os.Setenv("AUTH_USERNAME", "admin")
	os.Setenv("AUTH_PASSWORD", "secret")
	defer os.Unsetenv("AUTH_USERNAME")
	defer os.Unsetenv("AUTH_PASSWORD")

	handler := BasicAuth(okHandler)
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestBasicAuthWrongCreds(t *testing.T) {
	os.Setenv("AUTH_USERNAME", "admin")
	os.Setenv("AUTH_PASSWORD", "secret")
	defer os.Unsetenv("AUTH_USERNAME")
	defer os.Unsetenv("AUTH_PASSWORD")

	handler := BasicAuth(okHandler)
	req := httptest.NewRequest("GET", "/test", nil)
	req.SetBasicAuth("admin", "wrong")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestBasicAuthCorrectCreds(t *testing.T) {
	os.Setenv("AUTH_USERNAME", "admin")
	os.Setenv("AUTH_PASSWORD", "secret")
	defer os.Unsetenv("AUTH_USERNAME")
	defer os.Unsetenv("AUTH_PASSWORD")

	handler := BasicAuth(okHandler)
	req := httptest.NewRequest("GET", "/test", nil)
	req.SetBasicAuth("admin", "secret")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestApiKeyAuthNoHeader(t *testing.T) {
	os.Setenv("AUTH_APIKEY", "testkey123")
	defer os.Unsetenv("AUTH_APIKEY")

	handler := ApiKeyAuth(okHandler)
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestApiKeyAuthWrongFormat(t *testing.T) {
	os.Setenv("AUTH_APIKEY", "testkey123")
	defer os.Unsetenv("AUTH_APIKEY")

	handler := ApiKeyAuth(okHandler)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Basic dGVzdA==")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestApiKeyAuthWrongKey(t *testing.T) {
	os.Setenv("AUTH_APIKEY", "testkey123")
	defer os.Unsetenv("AUTH_APIKEY")

	handler := ApiKeyAuth(okHandler)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer wrongkey")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestApiKeyAuthCorrectKey(t *testing.T) {
	os.Setenv("AUTH_APIKEY", "testkey123")
	defer os.Unsetenv("AUTH_APIKEY")

	handler := ApiKeyAuth(okHandler)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer testkey123")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestAuthDispatcherBasic(t *testing.T) {
	os.Setenv("AUTH_TYPE", "basic")
	os.Setenv("AUTH_USERNAME", "admin")
	os.Setenv("AUTH_PASSWORD", "secret")
	defer os.Unsetenv("AUTH_TYPE")
	defer os.Unsetenv("AUTH_USERNAME")
	defer os.Unsetenv("AUTH_PASSWORD")

	handler := Auth(okHandler)
	req := httptest.NewRequest("GET", "/test", nil)
	req.SetBasicAuth("admin", "secret")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 with basic auth, got %d", rr.Code)
	}
}

func TestAuthDispatcherApiKey(t *testing.T) {
	os.Setenv("AUTH_TYPE", "apikey")
	os.Setenv("AUTH_APIKEY", "mykey")
	defer os.Unsetenv("AUTH_TYPE")
	defer os.Unsetenv("AUTH_APIKEY")

	handler := Auth(okHandler)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer mykey")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 with apikey auth, got %d", rr.Code)
	}
}
