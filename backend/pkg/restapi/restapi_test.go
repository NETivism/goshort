package restapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"

	"github.com/netivism/goshort/backend/pkg/db"
	"github.com/netivism/goshort/backend/pkg/model"
)

func setupTestDB(t *testing.T) {
	t.Helper()
	err := db.ConnectForTest(sqlite.Open("file::memory:?cache=shared"))
	if err != nil {
		t.Fatalf("failed to connect test db: %v", err)
	}
}

func setBasicAuth(t *testing.T) {
	t.Helper()
	os.Setenv("AUTH_TYPE", "basic")
	os.Setenv("AUTH_USERNAME", "admin")
	os.Setenv("AUTH_PASSWORD", "secret")
	t.Cleanup(func() {
		os.Unsetenv("AUTH_TYPE")
		os.Unsetenv("AUTH_USERNAME")
		os.Unsetenv("AUTH_PASSWORD")
	})
}

func setApiKeyAuth(t *testing.T) {
	t.Helper()
	os.Setenv("AUTH_TYPE", "apikey")
	os.Setenv("AUTH_APIKEY", "testkey")
	t.Cleanup(func() {
		os.Unsetenv("AUTH_TYPE")
		os.Unsetenv("AUTH_APIKEY")
	})
}

// --- handle/create tests ---

func TestCreateShortURL(t *testing.T) {
	setBasicAuth(t)
	setupTestDB(t)

	router := New()
	body := `{"redirect":"https://example.com/long-page"}`
	req := httptest.NewRequest("POST", "/handle/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d, body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["success"] != float64(1) {
		t.Fatalf("expected success=1, got %v", resp["success"])
	}

	results := resp["result"].([]interface{})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	entry := results[0].(map[string]interface{})
	shortID := entry["short"].(string)
	if shortID == "" {
		t.Fatal("expected non-empty short ID")
	}
	if len(shortID) < 5 {
		t.Errorf("expected short ID length >= 5, got %d (%s)", len(shortID), shortID)
	}
	if entry["redirect"] != "https://example.com/long-page" {
		t.Errorf("expected redirect=https://example.com/long-page, got %s", entry["redirect"])
	}

	// Verify database record
	dbi := db.Get()
	var record model.Redirect
	dbi.Where("id = ?", shortID).First(&record)
	if record.Redirect != "https://example.com/long-page" {
		t.Errorf("DB redirect mismatch: expected https://example.com/long-page, got %s", record.Redirect)
	}
	if record.Domain != "example.com" {
		t.Errorf("DB domain mismatch: expected example.com, got %s", record.Domain)
	}
	if record.Path != "/long-page" {
		t.Errorf("DB path mismatch: expected /long-page, got %s", record.Path)
	}
}

func TestCreateShortURLInvalidBody(t *testing.T) {
	setBasicAuth(t)
	setupTestDB(t)

	router := New()
	body := `{"redirect":""}`
	req := httptest.NewRequest("POST", "/handle/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty redirect, got %d", rr.Code)
	}
}

func TestCreateShortURLNonHTTP(t *testing.T) {
	setBasicAuth(t)
	setupTestDB(t)

	router := New()
	body := `{"redirect":"ftp://example.com/file"}`
	req := httptest.NewRequest("POST", "/handle/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for non-http scheme, got %d", rr.Code)
	}
}

// --- Full flow: create → redirect → verify visit ---

func TestCreateThenRedirectFlow(t *testing.T) {
	setBasicAuth(t)
	setupTestDB(t)
	router := New()

	// Step 1: Create a short URL
	body := `{"redirect":"https://target-site.org/landing?page=1"}`
	createReq := httptest.NewRequest("POST", "/handle/create", strings.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	router.ServeHTTP(createRR, createReq)

	if createRR.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d, body: %s", createRR.Code, createRR.Body.String())
	}

	var createResp map[string]interface{}
	json.NewDecoder(createRR.Body).Decode(&createResp)
	shortID := createResp["result"].([]interface{})[0].(map[string]interface{})["short"].(string)
	t.Logf("created short ID: %s", shortID)

	// Step 2: Visit the short URL with a Facebook referrer
	visitReq := httptest.NewRequest("GET", "/"+shortID, nil)
	visitReq.Header.Set("Referer", "https://www.facebook.com/post/456")
	visitReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
	visitRR := httptest.NewRecorder()
	router.ServeHTTP(visitRR, visitReq)

	if visitRR.Code != http.StatusMovedPermanently {
		t.Fatalf("redirect: expected 301, got %d", visitRR.Code)
	}
	location := visitRR.Header().Get("Location")
	if location != "https://target-site.org/landing?page=1" {
		t.Errorf("redirect location mismatch: got %s", location)
	}

	// Step 3: Verify visit record in database
	dbi := db.Get()
	var visits []model.Visits
	dbi.Where("redirect_id = ?", shortID).Find(&visits)
	if len(visits) != 1 {
		t.Fatalf("expected 1 visit record, got %d", len(visits))
	}
	v := visits[0]
	if v.RedirectId != shortID {
		t.Errorf("visit redirect_id mismatch: expected %s, got %s", shortID, v.RedirectId)
	}
	if v.Referer.Type != "social" {
		t.Errorf("expected referer_type=social, got %s", v.Referer.Type)
	}
	if v.Referer.Network != "facebook" {
		t.Errorf("expected referer_network=facebook, got %s", v.Referer.Network)
	}
	if v.CreatedAt == 0 {
		t.Error("expected non-zero created_at timestamp")
	}
}


func TestCreateThenRedirectWithGoogleSearch(t *testing.T) {
	setBasicAuth(t)
	setupTestDB(t)
	router := New()

	body := `{"redirect":"https://example.com/dest"}`
	createReq := httptest.NewRequest("POST", "/handle/create", strings.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	router.ServeHTTP(createRR, createReq)

	var createResp map[string]interface{}
	json.NewDecoder(createRR.Body).Decode(&createResp)
	shortID := createResp["result"].([]interface{})[0].(map[string]interface{})["short"].(string)

	// Visit with Google search referrer
	visitReq := httptest.NewRequest("GET", "/"+shortID, nil)
	visitReq.Header.Set("Referer", "https://www.google.com/search?q=test+query")
	visitRR := httptest.NewRecorder()
	router.ServeHTTP(visitRR, visitReq)

	if visitRR.Code != http.StatusMovedPermanently {
		t.Fatalf("expected 301, got %d", visitRR.Code)
	}

	dbi := db.Get()
	var visits []model.Visits
	dbi.Where("redirect_id = ?", shortID).Find(&visits)
	if len(visits) != 1 {
		t.Fatalf("expected 1 visit, got %d", len(visits))
	}
	if visits[0].Referer.Type != "search" {
		t.Errorf("expected referer_type=search, got %s", visits[0].Referer.Type)
	}
	if visits[0].Referer.Network != "google" {
		t.Errorf("expected referer_network=google, got %s", visits[0].Referer.Network)
	}
}

func TestCreateThenRedirectWithLineUA(t *testing.T) {
	setBasicAuth(t)
	setupTestDB(t)
	router := New()

	body := `{"redirect":"https://example.com/dest"}`
	createReq := httptest.NewRequest("POST", "/handle/create", strings.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	router.ServeHTTP(createRR, createReq)

	var createResp map[string]interface{}
	json.NewDecoder(createRR.Body).Decode(&createResp)
	shortID := createResp["result"].([]interface{})[0].(map[string]interface{})["short"].(string)

	// Visit with Line User-Agent
	visitReq := httptest.NewRequest("GET", "/"+shortID, nil)
	visitReq.Header.Set("User-Agent", "Mozilla/5.0 Line/8.12.0")
	visitRR := httptest.NewRecorder()
	router.ServeHTTP(visitRR, visitReq)

	if visitRR.Code != http.StatusMovedPermanently {
		t.Fatalf("expected 301, got %d", visitRR.Code)
	}

	dbi := db.Get()
	var visits []model.Visits
	dbi.Where("redirect_id = ?", shortID).Find(&visits)
	if len(visits) != 1 {
		t.Fatalf("expected 1 visit, got %d", len(visits))
	}
	if visits[0].Referer.Type != "social" {
		t.Errorf("expected referer_type=social, got %s", visits[0].Referer.Type)
	}
	if visits[0].Referer.Network != "line" {
		t.Errorf("expected referer_network=line, got %s", visits[0].Referer.Network)
	}
}

func TestCreateThenRedirectDirectTraffic(t *testing.T) {
	setBasicAuth(t)
	setupTestDB(t)
	router := New()

	body := `{"redirect":"https://example.com/dest"}`
	createReq := httptest.NewRequest("POST", "/handle/create", strings.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	router.ServeHTTP(createRR, createReq)

	var createResp map[string]interface{}
	json.NewDecoder(createRR.Body).Decode(&createResp)
	shortID := createResp["result"].([]interface{})[0].(map[string]interface{})["short"].(string)

	// Visit with no Referer header (direct traffic)
	visitReq := httptest.NewRequest("GET", "/"+shortID, nil)
	visitRR := httptest.NewRecorder()
	router.ServeHTTP(visitRR, visitReq)

	if visitRR.Code != http.StatusMovedPermanently {
		t.Fatalf("expected 301, got %d", visitRR.Code)
	}

	dbi := db.Get()
	var visits []model.Visits
	dbi.Where("redirect_id = ?", shortID).Find(&visits)
	if len(visits) != 1 {
		t.Fatalf("expected 1 visit, got %d", len(visits))
	}
	if visits[0].Referer.Type != "direct" {
		t.Errorf("expected referer_type=direct, got %s", visits[0].Referer.Type)
	}
	if visits[0].Referer.Network != "" {
		t.Errorf("expected empty referer_network, got %s", visits[0].Referer.Network)
	}
}

func TestCreateThenMultipleVisits(t *testing.T) {
	setBasicAuth(t)
	setupTestDB(t)
	router := New()

	body := `{"redirect":"https://example.com/multi"}`
	createReq := httptest.NewRequest("POST", "/handle/create", strings.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	router.ServeHTTP(createRR, createReq)

	var createResp map[string]interface{}
	json.NewDecoder(createRR.Body).Decode(&createResp)
	shortID := createResp["result"].([]interface{})[0].(map[string]interface{})["short"].(string)

	// Visit 1: Facebook
	r1 := httptest.NewRequest("GET", "/"+shortID, nil)
	r1.Header.Set("Referer", "https://www.facebook.com/")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, r1)

	// Visit 2: Google search
	r2 := httptest.NewRequest("GET", "/"+shortID, nil)
	r2.Header.Set("Referer", "https://www.google.com/search?q=test")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)

	// Visit 3: Direct
	r3 := httptest.NewRequest("GET", "/"+shortID, nil)
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, r3)

	// Verify 3 visit records
	dbi := db.Get()
	var visits []model.Visits
	dbi.Where("redirect_id = ?", shortID).Order("id").Find(&visits)
	if len(visits) != 3 {
		t.Fatalf("expected 3 visits, got %d", len(visits))
	}
	if visits[0].Referer.Type != "social" || visits[0].Referer.Network != "facebook" {
		t.Errorf("visit 1: expected social/facebook, got %s/%s", visits[0].Referer.Type, visits[0].Referer.Network)
	}
	if visits[1].Referer.Type != "search" || visits[1].Referer.Network != "google" {
		t.Errorf("visit 2: expected search/google, got %s/%s", visits[1].Referer.Type, visits[1].Referer.Network)
	}
	if visits[2].Referer.Type != "direct" {
		t.Errorf("visit 3: expected direct, got %s", visits[2].Referer.Type)
	}

	// Verify via visits API
	visitsReq := httptest.NewRequest("GET", "/handle/visits/"+shortID, nil)
	visitsReq.SetBasicAuth("admin", "secret")
	visitsRR := httptest.NewRecorder()
	router.ServeHTTP(visitsRR, visitsReq)

	if visitsRR.Code != http.StatusOK {
		t.Fatalf("visits API: expected 200, got %d", visitsRR.Code)
	}
	var visitsResp map[string]interface{}
	json.NewDecoder(visitsRR.Body).Decode(&visitsResp)
	resultObj := visitsResp["result"].(map[string]interface{})
	refStats := resultObj["referrer_statistics"].(map[string]interface{})
	if len(refStats) != 3 {
		t.Errorf("visits API: expected 3 referrer types, got %d", len(refStats))
	}
	dateStats := resultObj["dates"].(map[string]interface{})
	if len(dateStats) == 0 {
		t.Errorf("visits API: expected at least 1 date entry")
	}
}

// --- /{id} redirect edge cases ---

func TestRedirectInvalidID(t *testing.T) {
	setBasicAuth(t)
	setupTestDB(t)
	router := New()

	req := httptest.NewRequest("GET", "/invalid-id!", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for invalid ID chars, got %d", rr.Code)
	}
}

func TestRedirectNonexistentID(t *testing.T) {
	setBasicAuth(t)
	setupTestDB(t)
	router := New()

	// Valid format but doesn't exist — Root handler finds no record
	// and does nothing (no redirect, no response body written by handler)
	req := httptest.NewRequest("GET", "/zzzzz", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// No redirect should occur
	if rr.Code == http.StatusMovedPermanently {
		t.Error("should not redirect for nonexistent ID")
	}
}

// --- /handle/visits tests ---

func TestVisitsWithBasicAuth(t *testing.T) {
	setBasicAuth(t)
	setupTestDB(t)

	dbi := db.Get()
	dbi.Create(&model.Redirect{
		Id: "vis01", Redirect: "https://example.com/target", Domain: "example.com", Path: "/target",
	})
	dbi.Create(&model.Visits{
		RedirectId: "vis01",
		Referer:    model.Referer{Type: "social", Network: "facebook"},
	})
	dbi.Create(&model.Visits{
		RedirectId: "vis01",
		Referer:    model.Referer{Type: "direct"},
	})

	router := New()
	req := httptest.NewRequest("GET", "/handle/visits/vis01", nil)
	req.SetBasicAuth("admin", "secret")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["success"] != float64(1) {
		t.Fatalf("expected success=1, got %v", resp["success"])
	}
	resultObj := resp["result"].(map[string]interface{})
	refStats := resultObj["referrer_statistics"].(map[string]interface{})
	if len(refStats) != 2 {
		t.Errorf("expected 2 referrer types (social, direct), got %d", len(refStats))
	}
	social := refStats["social"].(map[string]interface{})
	if social["all"] != float64(1) {
		t.Errorf("expected social.all=1, got %v", social["all"])
	}
	direct := refStats["direct"].(map[string]interface{})
	if direct["all"] != float64(1) {
		t.Errorf("expected direct.all=1, got %v", direct["all"])
	}
}

func TestVisitsWithApiKeyAuth(t *testing.T) {
	setApiKeyAuth(t)
	setupTestDB(t)

	dbi := db.Get()
	dbi.Create(&model.Redirect{
		Id: "vis02", Redirect: "https://example.com/target", Domain: "example.com", Path: "/target",
	})
	dbi.Create(&model.Visits{
		RedirectId: "vis02",
		Referer:    model.Referer{Type: "search", Network: "google"},
	})

	router := New()
	req := httptest.NewRequest("GET", "/handle/visits/vis02", nil)
	req.Header.Set("Authorization", "Bearer testkey")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	resultObj := resp["result"].(map[string]interface{})
	refStats := resultObj["referrer_statistics"].(map[string]interface{})
	search := refStats["search"].(map[string]interface{})
	if search["all"] != float64(1) {
		t.Errorf("expected search.all=1, got %v", search["all"])
	}
	if search["google"] != float64(1) {
		t.Errorf("expected search.google=1, got %v", search["google"])
	}
}

func TestVisitsWithoutAuth(t *testing.T) {
	setBasicAuth(t)
	setupTestDB(t)

	router := New()
	req := httptest.NewRequest("GET", "/handle/visits/abc123", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestVisitsNonexistent(t *testing.T) {
	setApiKeyAuth(t)
	setupTestDB(t)

	router := New()
	req := httptest.NewRequest("GET", "/handle/visits/nonexistent", nil)
	req.Header.Set("Authorization", "Bearer testkey")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["message"] != "No visits found." {
		t.Errorf("expected 'No visits found.' message, got %v", resp["message"])
	}
	resultObj := resp["result"].(map[string]interface{})
	refStats := resultObj["referrer_statistics"].(map[string]interface{})
	if len(refStats) != 0 {
		t.Errorf("expected empty referrer_statistics, got %d entries", len(refStats))
	}
	dates := resultObj["dates"].(map[string]interface{})
	if len(dates) != 0 {
		t.Errorf("expected empty dates, got %d entries", len(dates))
	}
}

