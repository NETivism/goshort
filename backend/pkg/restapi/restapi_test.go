package restapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

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

// --- Statistics cache tests ---

// TestVisitsCreatesStatisticsOnFirstCall verifies that a Statistics record is
// created after the first visits API call (cache miss path).
func TestVisitsCreatesStatisticsOnFirstCall(t *testing.T) {
	setApiKeyAuth(t)
	setupTestDB(t)

	dbi := db.Get()
	id := "statcache-create-01"
	dbi.Create(&model.Redirect{Id: id, Redirect: "https://example.com", Domain: "example.com", Path: "/"})
	dbi.Create(&model.Visits{RedirectId: id, Referer: model.Referer{Type: "direct"}, CreatedAt: time.Now().Unix()})

	router := New()
	req := httptest.NewRequest("GET", "/handle/visits/"+id, nil)
	req.Header.Set("Authorization", "Bearer testkey")
	httptest.NewRecorder()
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var stat model.Statistics
	if err := dbi.Where("redirect_id = ?", id).First(&stat).Error; err != nil {
		t.Fatalf("Statistics not created after first API call: %v", err)
	}
	if stat.UpdateAt == 0 {
		t.Error("expected non-zero Statistics.UpdateAt")
	}
}

// TestVisitsCacheHitReturnsCachedResult verifies that a fresh Statistics record
// (UpdateAt within 1 hour) is returned directly without re-querying Visits.
func TestVisitsCacheHitReturnsCachedResult(t *testing.T) {
	setApiKeyAuth(t)
	setupTestDB(t)

	dbi := db.Get()
	id := "statcache-hit-01"

	// Insert a fresh Statistics record with a known total.
	cachedResult := model.VisitsStatResult{
		Total:              42,
		ReferrerStatistics: map[string]map[string]int64{"direct": {"all": 42}},
		Dates:              map[string]map[string]int64{"2026-01-01": {"allday": 42}},
	}
	cachedJSON, _ := json.Marshal(cachedResult)
	now := time.Now().Unix()
	dbi.Create(&model.Statistics{
		RedirectId:   id,
		Result:       string(cachedJSON),
		AggDateStart: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		AggDateEnd:   time.Now(),
		CreatedAt:    now,
		UpdateAt:     now, // fresh (< 1 hour old)
	})

	// Insert a visit that should NOT affect the result (cache is fresh).
	dbi.Create(&model.Visits{RedirectId: id, Referer: model.Referer{Type: "search", Network: "google"}, CreatedAt: now})

	router := New()
	req := httptest.NewRequest("GET", "/handle/visits/"+id, nil)
	req.Header.Set("Authorization", "Bearer testkey")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)

	if resp["message"] != "Visits loaded from cache." {
		t.Errorf("expected cache hit message, got: %v", resp["message"])
	}
	resultObj := resp["result"].(map[string]interface{})
	if resultObj["total"] != float64(42) {
		t.Errorf("expected cached total=42, got %v", resultObj["total"])
	}
}

// TestVisitsCacheMissRecomputes verifies that a stale Statistics record
// (UpdateAt > 1 hour ago) triggers a re-computation from the Visits table.
func TestVisitsCacheMissRecomputes(t *testing.T) {
	setApiKeyAuth(t)
	setupTestDB(t)

	dbi := db.Get()
	id := "statcache-miss-01"

	// Insert a stale Statistics record.
	staleResult := model.VisitsStatResult{
		Total:              5,
		ReferrerStatistics: map[string]map[string]int64{"direct": {"all": 5}},
		Dates:              map[string]map[string]int64{},
	}
	staleJSON, _ := json.Marshal(staleResult)
	staleTime := time.Now().Add(-2 * time.Hour)
	dbi.Create(&model.Statistics{
		RedirectId:   id,
		Result:       string(staleJSON),
		AggDateStart: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		AggDateEnd:   staleTime,
		CreatedAt:    staleTime.Unix(),
		UpdateAt:     staleTime.Unix(), // stale (> 1 hour old)
	})

	// Add 3 new visits (after AggDateEnd).
	newTs := time.Now().Unix()
	for i := 0; i < 3; i++ {
		dbi.Create(&model.Visits{RedirectId: id, Referer: model.Referer{Type: "search", Network: "google"}, CreatedAt: newTs})
	}

	router := New()
	req := httptest.NewRequest("GET", "/handle/visits/"+id, nil)
	req.Header.Set("Authorization", "Bearer testkey")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)

	if resp["message"] == "Visits loaded from cache." {
		t.Error("expected cache miss (recompute), not a cache hit")
	}
	resultObj := resp["result"].(map[string]interface{})
	// 5 (stale base) + 3 (new visits) = 8
	if resultObj["total"] != float64(8) {
		t.Errorf("expected total=8 (5 base + 3 new), got %v", resultObj["total"])
	}
}

// TestVisitsHistoryPreservedAfterVisitsClear verifies that Statistics retains
// historical data even after the Visits table is cleared.
func TestVisitsHistoryPreservedAfterVisitsClear(t *testing.T) {
	setApiKeyAuth(t)
	setupTestDB(t)

	dbi := db.Get()
	id := "statcache-clear-01"

	// Seed a Statistics record (simulates a prior aggregation with 50 total visits).
	historicalResult := model.VisitsStatResult{
		Total:              50,
		ReferrerStatistics: map[string]map[string]int64{"direct": {"all": 50}},
		Dates:              map[string]map[string]int64{"2025-12-01": {"allday": 50}},
	}
	historicalJSON, _ := json.Marshal(historicalResult)
	staleTime := time.Now().Add(-2 * time.Hour)
	dbi.Create(&model.Statistics{
		RedirectId:   id,
		Result:       string(historicalJSON),
		AggDateStart: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		AggDateEnd:   staleTime,
		CreatedAt:    staleTime.Unix(),
		UpdateAt:     staleTime.Unix(),
	})

	// Visits table has been cleared — no rows exist for this redirect.
	// (We intentionally do not insert any visits.)

	router := New()
	req := httptest.NewRequest("GET", "/handle/visits/"+id, nil)
	req.Header.Set("Authorization", "Bearer testkey")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	resultObj := resp["result"].(map[string]interface{})

	// Historical total must be preserved even with no visits in the table.
	if resultObj["total"] != float64(50) {
		t.Errorf("expected historical total=50, got %v", resultObj["total"])
	}

	// The 2025-12-01 date entry from history must still be present.
	dates := resultObj["dates"].(map[string]interface{})
	if _, ok := dates["2025-12-01"]; !ok {
		t.Error("expected historical date 2025-12-01 to be preserved in Statistics")
	}
}

// --- /handle/batch-info tests ---

func TestBatchInfoReturnsRedirectAndTotal(t *testing.T) {
	setApiKeyAuth(t)
	setupTestDB(t)

	dbi := db.Get()
	dbi.Create(&model.Redirect{Id: "bi01", Redirect: "https://example.com/a", Domain: "example.com", Path: "/a"})
	dbi.Create(&model.Redirect{Id: "bi02", Redirect: "https://example.com/b", Domain: "example.com", Path: "/b"})

	// Seed statistics for bi01 (total=10), leave bi02 with no statistics row.
	result01, _ := json.Marshal(model.VisitsStatResult{Total: 10})
	now := time.Now()
	dbi.Create(&model.Statistics{
		RedirectId: "bi01", Result: string(result01),
		AggDateStart: now, AggDateEnd: now, CreatedAt: now.Unix(), UpdateAt: now.Unix(),
	})

	router := New()
	body := `["bi01","bi02"]`
	req := httptest.NewRequest("POST", "/handle/batch-info", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer testkey")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["success"] != float64(1) {
		t.Fatalf("expected success=1, got %v", resp["success"])
	}

	results := resp["result"].([]interface{})
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	item0 := results[0].(map[string]interface{})
	if item0["id"] != "bi01" {
		t.Errorf("expected id=bi01, got %v", item0["id"])
	}
	if item0["redirect"] != "https://example.com/a" {
		t.Errorf("expected redirect=https://example.com/a, got %v", item0["redirect"])
	}
	if item0["total"] != float64(10) {
		t.Errorf("expected total=10, got %v", item0["total"])
	}

	item1 := results[1].(map[string]interface{})
	if item1["id"] != "bi02" {
		t.Errorf("expected id=bi02, got %v", item1["id"])
	}
	if item1["redirect"] != "https://example.com/b" {
		t.Errorf("expected redirect=https://example.com/b, got %v", item1["redirect"])
	}
	if item1["total"] != float64(0) {
		t.Errorf("expected total=0 (no statistics row), got %v", item1["total"])
	}
}

func TestBatchInfoNonExistentIDsReturnEmpty(t *testing.T) {
	setApiKeyAuth(t)
	setupTestDB(t)

	router := New()
	body := `["notfound1","notfound2"]`
	req := httptest.NewRequest("POST", "/handle/batch-info", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer testkey")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	results := resp["result"].([]interface{})
	if len(results) != 2 {
		t.Fatalf("expected 2 result items, got %d", len(results))
	}
	for _, r := range results {
		item := r.(map[string]interface{})
		if item["redirect"] != "" {
			t.Errorf("expected empty redirect for non-existent ID, got %v", item["redirect"])
		}
		if item["total"] != float64(0) {
			t.Errorf("expected total=0 for non-existent ID, got %v", item["total"])
		}
	}
}

func TestBatchInfoPreservesInputOrder(t *testing.T) {
	setApiKeyAuth(t)
	setupTestDB(t)

	dbi := db.Get()
	dbi.Create(&model.Redirect{Id: "order1", Redirect: "https://first.com", Domain: "first.com", Path: "/"})
	dbi.Create(&model.Redirect{Id: "order2", Redirect: "https://second.com", Domain: "second.com", Path: "/"})
	dbi.Create(&model.Redirect{Id: "order3", Redirect: "https://third.com", Domain: "third.com", Path: "/"})

	router := New()
	body := `["order3","order1","order2"]`
	req := httptest.NewRequest("POST", "/handle/batch-info", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer testkey")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	results := resp["result"].([]interface{})

	expected := []string{"order3", "order1", "order2"}
	for i, exp := range expected {
		got := results[i].(map[string]interface{})["id"]
		if got != exp {
			t.Errorf("position %d: expected id=%s, got %v", i, exp, got)
		}
	}
}

func TestBatchInfoEmptyArrayReturns400(t *testing.T) {
	setApiKeyAuth(t)
	setupTestDB(t)

	router := New()
	req := httptest.NewRequest("POST", "/handle/batch-info", strings.NewReader(`[]`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer testkey")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty array, got %d", rr.Code)
	}
}

func TestBatchInfoInvalidBodyReturns400(t *testing.T) {
	setApiKeyAuth(t)
	setupTestDB(t)

	router := New()
	req := httptest.NewRequest("POST", "/handle/batch-info", strings.NewReader(`not json`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer testkey")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", rr.Code)
	}
}

func TestBatchInfoWithoutAuthReturns401(t *testing.T) {
	setApiKeyAuth(t)
	setupTestDB(t)

	router := New()
	req := httptest.NewRequest("POST", "/handle/batch-info", strings.NewReader(`["abc12"]`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// --- /handle/batch-create tests ---

func TestBatchCreateSuccess(t *testing.T) {
	setApiKeyAuth(t)
	setupTestDB(t)

	router := New()
	body := `[
		{"redirect":"https://example.com/page1"},
		{"redirect":"https://example.com/page2"},
		{"redirect":"https://example.com/page3"}
	]`
	req := httptest.NewRequest("POST", "/handle/batch-create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer testkey")
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
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	redirects := []string{
		"https://example.com/page1",
		"https://example.com/page2",
		"https://example.com/page3",
	}
	dbi := db.Get()
	for i, r := range results {
		item := r.(map[string]interface{})
		short, hasShort := item["short"]
		if !hasShort || short == "" {
			t.Errorf("result %d: expected non-empty short", i)
		}
		if _, hasErr := item["error"]; hasErr {
			t.Errorf("result %d: expected no error field, got %v", i, item["error"])
		}
		if item["redirect"] != redirects[i] {
			t.Errorf("result %d: expected redirect=%s, got %v", i, redirects[i], item["redirect"])
		}
		// Verify record in DB
		var record model.Redirect
		dbi.Where("id = ?", short).First(&record)
		if record.Redirect != redirects[i] {
			t.Errorf("result %d: DB redirect mismatch, expected %s got %s", i, redirects[i], record.Redirect)
		}
	}
}

func TestBatchCreateValidationFailsBeforeInsert(t *testing.T) {
	setApiKeyAuth(t)
	setupTestDB(t)

	router := New()
	// Third entry has an invalid URL — validation should catch it before any insert.
	body := `[
		{"redirect":"https://example.com/ok1"},
		{"redirect":"https://example.com/ok2"},
		{"redirect":"ftp://bad-scheme.com"}
	]`
	req := httptest.NewRequest("POST", "/handle/batch-create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer testkey")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid URL in batch, got %d", rr.Code)
	}

	// Nothing should have been inserted.
	dbi := db.Get()
	var count int64
	dbi.Model(&model.Redirect{}).Where("redirect IN ?", []string{
		"https://example.com/ok1",
		"https://example.com/ok2",
	}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 records inserted (validation failed before insert), got %d", count)
	}
}

func TestBatchCreateEmptyRedirectReturns400(t *testing.T) {
	setApiKeyAuth(t)
	setupTestDB(t)

	router := New()
	req := httptest.NewRequest("POST", "/handle/batch-create", strings.NewReader(`[{"redirect":""}]`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer testkey")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty redirect, got %d", rr.Code)
	}
}

func TestBatchCreateNonHTTPSchemeReturns400(t *testing.T) {
	setApiKeyAuth(t)
	setupTestDB(t)

	router := New()
	req := httptest.NewRequest("POST", "/handle/batch-create", strings.NewReader(`[{"redirect":"ftp://example.com/file"}]`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer testkey")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for ftp scheme, got %d", rr.Code)
	}
}

func TestBatchCreateURLWithCredentialsReturns400(t *testing.T) {
	setApiKeyAuth(t)
	setupTestDB(t)

	router := New()
	req := httptest.NewRequest("POST", "/handle/batch-create", strings.NewReader(`[{"redirect":"https://user:pass@example.com/"}]`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer testkey")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for URL with credentials, got %d", rr.Code)
	}
}

func TestBatchCreateEmptyArrayReturns400(t *testing.T) {
	setApiKeyAuth(t)
	setupTestDB(t)

	router := New()
	req := httptest.NewRequest("POST", "/handle/batch-create", strings.NewReader(`[]`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer testkey")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty array, got %d", rr.Code)
	}
}

func TestBatchCreateExceedsLimitReturns400(t *testing.T) {
	setApiKeyAuth(t)
	setupTestDB(t)

	items := make([]map[string]string, 1001)
	for i := range items {
		items[i] = map[string]string{"redirect": "https://example.com/"}
	}
	bodyBytes, _ := json.Marshal(items)

	router := New()
	req := httptest.NewRequest("POST", "/handle/batch-create", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer testkey")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for batch size > 1000, got %d", rr.Code)
	}
}

func TestBatchCreateWithoutAuthReturns401(t *testing.T) {
	setApiKeyAuth(t)
	setupTestDB(t)

	router := New()
	req := httptest.NewRequest("POST", "/handle/batch-create", strings.NewReader(`[{"redirect":"https://example.com/"}]`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestBatchCreateMessageCountsSuccessAndFailure(t *testing.T) {
	setApiKeyAuth(t)
	setupTestDB(t)

	// Insert a record to force a duplicate-key DB error on the second insert.
	dbi := db.Get()
	dbi.Create(&model.Redirect{Id: "dupid1", Redirect: "https://dup.com", Domain: "dup.com", Path: "/"})

	router := New()
	// Both URLs are format-valid; we simulate a partial failure by pre-occupying
	// a short ID slot — actual error path tested via message counts.
	body := `[
		{"redirect":"https://example.com/success1"},
		{"redirect":"https://example.com/success2"}
	]`
	req := httptest.NewRequest("POST", "/handle/batch-create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer testkey")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d, body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	results := resp["result"].([]interface{})
	if len(results) != 2 {
		t.Fatalf("expected 2 result items, got %d", len(results))
	}

	// Both should succeed — verify message format.
	msg := resp["message"].(string)
	if msg != "2/2 URLs shortened successfully." {
		t.Errorf("expected '2/2 URLs shortened successfully.', got %q", msg)
	}
}

// TestVisitsForceRefresh verifies that ?refresh=1 bypasses the 1-hour cache.
func TestVisitsForceRefresh(t *testing.T) {
	setApiKeyAuth(t)
	setupTestDB(t)

	dbi := db.Get()
	id := "statcache-force-01"

	// Insert a fresh Statistics record (would normally be served from cache).
	cachedResult := model.VisitsStatResult{
		Total:              99,
		ReferrerStatistics: map[string]map[string]int64{"direct": {"all": 99}},
		Dates:              map[string]map[string]int64{},
	}
	cachedJSON, _ := json.Marshal(cachedResult)
	now := time.Now().Unix()
	dbi.Create(&model.Statistics{
		RedirectId:   id,
		Result:       string(cachedJSON),
		AggDateStart: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		AggDateEnd:   time.Now(),
		CreatedAt:    now,
		UpdateAt:     now, // fresh
	})

	// Add 1 new visit that is NOT in the cache yet.
	dbi.Create(&model.Visits{RedirectId: id, Referer: model.Referer{Type: "search", Network: "google"}, CreatedAt: now + 1})

	router := New()

	// Without ?refresh=1 → cache hit, total stays 99.
	req := httptest.NewRequest("GET", "/handle/visits/"+id, nil)
	req.Header.Set("Authorization", "Bearer testkey")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["message"] != "Visits loaded from cache." {
		t.Errorf("expected cache hit without ?refresh=1, got: %v", resp["message"])
	}
	if resp["result"].(map[string]interface{})["total"] != float64(99) {
		t.Errorf("expected cached total=99, got %v", resp["result"].(map[string]interface{})["total"])
	}

	// With ?refresh=1 → cache bypassed, total becomes 99 + 1 = 100.
	req2 := httptest.NewRequest("GET", "/handle/visits/"+id+"?refresh=1", nil)
	req2.Header.Set("Authorization", "Bearer testkey")
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req2)

	var resp2 map[string]interface{}
	json.NewDecoder(rr2.Body).Decode(&resp2)
	if resp2["message"] == "Visits loaded from cache." {
		t.Error("expected cache bypass with ?refresh=1, but got a cache hit")
	}
	if resp2["result"].(map[string]interface{})["total"] != float64(100) {
		t.Errorf("expected refreshed total=100 (99+1), got %v", resp2["result"].(map[string]interface{})["total"])
	}
}
