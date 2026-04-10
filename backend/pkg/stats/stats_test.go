package stats

import (
	"encoding/json"
	"testing"
	"time"

	"gorm.io/driver/sqlite"

	"github.com/netivism/goshort/backend/pkg/db"
	"github.com/netivism/goshort/backend/pkg/model"
)

func setupTestDB(t *testing.T) {
	t.Helper()
	if err := db.ConnectForTest(sqlite.Open("file::memory:?cache=shared")); err != nil {
		t.Fatalf("failed to connect test db: %v", err)
	}
}

// --- ComputeVisitStats ---

func TestComputeVisitStatsEmpty(t *testing.T) {
	result := ComputeVisitStats(nil)
	if result.Total != 0 {
		t.Errorf("expected total=0, got %d", result.Total)
	}
	if len(result.ReferrerStatistics) != 0 {
		t.Errorf("expected empty referrer_statistics")
	}
	if len(result.Dates) != 0 {
		t.Errorf("expected empty dates")
	}
}

func TestComputeVisitStatsGroupsByReferrer(t *testing.T) {
	ts := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC).Unix()
	visits := []model.Visits{
		{RedirectId: "x", Referer: model.Referer{Type: "social", Network: "facebook"}, CreatedAt: ts},
		{RedirectId: "x", Referer: model.Referer{Type: "social", Network: "facebook"}, CreatedAt: ts},
		{RedirectId: "x", Referer: model.Referer{Type: "search", Network: "google"}, CreatedAt: ts},
		{RedirectId: "x", Referer: model.Referer{Type: "", Network: ""}, CreatedAt: ts},
	}
	result := ComputeVisitStats(visits)

	if result.Total != 4 {
		t.Errorf("expected total=4, got %d", result.Total)
	}
	if result.ReferrerStatistics["social"]["all"] != 2 {
		t.Errorf("expected social.all=2, got %d", result.ReferrerStatistics["social"]["all"])
	}
	if result.ReferrerStatistics["social"]["facebook"] != 2 {
		t.Errorf("expected social.facebook=2, got %d", result.ReferrerStatistics["social"]["facebook"])
	}
	if result.ReferrerStatistics["search"]["all"] != 1 {
		t.Errorf("expected search.all=1, got %d", result.ReferrerStatistics["search"]["all"])
	}
	if result.ReferrerStatistics["search"]["google"] != 1 {
		t.Errorf("expected search.google=1, got %d", result.ReferrerStatistics["search"]["google"])
	}
	// empty type → "unknown"
	if result.ReferrerStatistics["unknown"]["all"] != 1 {
		t.Errorf("expected unknown.all=1, got %d", result.ReferrerStatistics["unknown"]["all"])
	}
}

func TestComputeVisitStatsGroupsByDate(t *testing.T) {
	day1 := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC).Unix()  // hour 9
	day2 := time.Date(2026, 3, 2, 14, 0, 0, 0, time.UTC).Unix() // hour 14
	visits := []model.Visits{
		{RedirectId: "x", Referer: model.Referer{Type: "direct"}, CreatedAt: day1},
		{RedirectId: "x", Referer: model.Referer{Type: "direct"}, CreatedAt: day1},
		{RedirectId: "x", Referer: model.Referer{Type: "direct"}, CreatedAt: day2},
	}
	result := ComputeVisitStats(visits)

	if result.Dates["2026-03-01"]["allday"] != 2 {
		t.Errorf("expected 2026-03-01 allday=2, got %d", result.Dates["2026-03-01"]["allday"])
	}
	if result.Dates["2026-03-02"]["allday"] != 1 {
		t.Errorf("expected 2026-03-02 allday=1, got %d", result.Dates["2026-03-02"]["allday"])
	}

	// Always stores all 24 hours regardless of threshold.
	if len(result.Dates["2026-03-01"]) != 25 { // allday + 24 hours
		t.Errorf("expected 25 keys (allday + 0~23) for 2026-03-01, got %d", len(result.Dates["2026-03-01"]))
	}
	if result.Dates["2026-03-01"]["9"] != 2 {
		t.Errorf("expected 2026-03-01 hour9=2, got %d", result.Dates["2026-03-01"]["9"])
	}
	if result.Dates["2026-03-01"]["10"] != 0 {
		t.Errorf("expected 2026-03-01 hour10=0 (stored as zero), got %d", result.Dates["2026-03-01"]["10"])
	}
	if result.Dates["2026-03-02"]["14"] != 1 {
		t.Errorf("expected 2026-03-02 hour14=1, got %d", result.Dates["2026-03-02"]["14"])
	}
}

// --- MergeVisitStats ---

func TestMergeVisitStatsBothEmpty(t *testing.T) {
	result := MergeVisitStats(model.VisitsStatResult{}, model.VisitsStatResult{})
	if result.Total != 0 {
		t.Errorf("expected total=0, got %d", result.Total)
	}
}

func TestMergeVisitStatsSumsTotals(t *testing.T) {
	base := model.VisitsStatResult{Total: 10}
	delta := model.VisitsStatResult{Total: 5}
	result := MergeVisitStats(base, delta)
	if result.Total != 15 {
		t.Errorf("expected total=15, got %d", result.Total)
	}
}

func TestMergeVisitStatsCombinesReferrers(t *testing.T) {
	base := model.VisitsStatResult{
		Total: 3,
		ReferrerStatistics: map[string]map[string]int64{
			"social": {"all": 3, "facebook": 2, "line": 1},
		},
		Dates: map[string]map[string]int64{},
	}
	delta := model.VisitsStatResult{
		Total: 2,
		ReferrerStatistics: map[string]map[string]int64{
			"social": {"all": 1, "facebook": 1},
			"search": {"all": 1, "google": 1},
		},
		Dates: map[string]map[string]int64{},
	}
	result := MergeVisitStats(base, delta)

	if result.Total != 5 {
		t.Errorf("expected total=5, got %d", result.Total)
	}
	if result.ReferrerStatistics["social"]["all"] != 4 {
		t.Errorf("expected social.all=4, got %d", result.ReferrerStatistics["social"]["all"])
	}
	if result.ReferrerStatistics["social"]["facebook"] != 3 {
		t.Errorf("expected social.facebook=3, got %d", result.ReferrerStatistics["social"]["facebook"])
	}
	if result.ReferrerStatistics["social"]["line"] != 1 {
		t.Errorf("expected social.line=1, got %d", result.ReferrerStatistics["social"]["line"])
	}
	if result.ReferrerStatistics["search"]["all"] != 1 {
		t.Errorf("expected search.all=1, got %d", result.ReferrerStatistics["search"]["all"])
	}
}

func TestMergeVisitStatsCombinesDates(t *testing.T) {
	base := model.VisitsStatResult{
		Total:              2,
		ReferrerStatistics: map[string]map[string]int64{},
		Dates: map[string]map[string]int64{
			"2026-01-01": {"allday": 2, "9": 2},
			"2026-01-02": {"allday": 1},
		},
	}
	delta := model.VisitsStatResult{
		Total:              3,
		ReferrerStatistics: map[string]map[string]int64{},
		Dates: map[string]map[string]int64{
			"2026-01-01": {"allday": 1, "10": 1},
			"2026-01-03": {"allday": 3},
		},
	}
	result := MergeVisitStats(base, delta)

	// Same date: counts are summed
	if result.Dates["2026-01-01"]["allday"] != 3 {
		t.Errorf("expected 2026-01-01 allday=3, got %d", result.Dates["2026-01-01"]["allday"])
	}
	if result.Dates["2026-01-01"]["9"] != 2 {
		t.Errorf("expected 2026-01-01 hour9=2, got %d", result.Dates["2026-01-01"]["9"])
	}
	if result.Dates["2026-01-01"]["10"] != 1 {
		t.Errorf("expected 2026-01-01 hour10=1, got %d", result.Dates["2026-01-01"]["10"])
	}
	// Base-only date preserved
	if result.Dates["2026-01-02"]["allday"] != 1 {
		t.Errorf("expected 2026-01-02 allday=1, got %d", result.Dates["2026-01-02"]["allday"])
	}
	// Delta-only date added
	if result.Dates["2026-01-03"]["allday"] != 3 {
		t.Errorf("expected 2026-01-03 allday=3, got %d", result.Dates["2026-01-03"]["allday"])
	}
}

// --- Refresh ---

func TestRefreshCreatesStatisticsOnFirstCall(t *testing.T) {
	setupTestDB(t)
	dbi := db.Get()
	id := "refresh-new-01"

	ts := time.Date(2026, 2, 10, 8, 0, 0, 0, time.UTC).Unix()
	dbi.Create(&model.Visits{RedirectId: id, Referer: model.Referer{Type: "direct"}, CreatedAt: ts})
	dbi.Create(&model.Visits{RedirectId: id, Referer: model.Referer{Type: "search", Network: "google"}, CreatedAt: ts})

	result, err := Refresh(dbi, id)
	if err != nil {
		t.Fatalf("Refresh error: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("expected total=2, got %d", result.Total)
	}

	// Statistics record must be created
	var stat model.Statistics
	if err := dbi.Where("redirect_id = ?", id).First(&stat).Error; err != nil {
		t.Fatalf("Statistics record not created: %v", err)
	}
	if stat.UpdateAt == 0 {
		t.Error("expected non-zero UpdateAt")
	}
	// AggDateStart must be set to approximately now (within a few seconds).
	if time.Since(stat.AggDateStart) > 5*time.Second {
		t.Errorf("expected AggDateStart to be approximately now, got %s", stat.AggDateStart)
	}
}

func TestRefreshIsIncremental(t *testing.T) {
	setupTestDB(t)
	dbi := db.Get()
	id := "refresh-incr-01"

	// Seed an existing Statistics record as if it was computed 2 hours ago.
	baseResult := model.VisitsStatResult{
		Total:              7,
		ReferrerStatistics: map[string]map[string]int64{"direct": {"all": 7}},
		Dates:              map[string]map[string]int64{"2026-01-01": {"allday": 7}},
	}
	baseJSON, _ := json.Marshal(baseResult)
	aggEnd := time.Now().Add(-2 * time.Hour)
	dbi.Create(&model.Statistics{
		RedirectId:   id,
		Result:       string(baseJSON),
		AggDateStart: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		AggDateEnd:   aggEnd,
		CreatedAt:    aggEnd.Unix(),
		UpdateAt:     aggEnd.Unix(),
	})

	// Old visits (before AggDateEnd) — must NOT be double-counted.
	oldTs := aggEnd.Add(-1 * time.Hour).Unix()
	dbi.Create(&model.Visits{RedirectId: id, Referer: model.Referer{Type: "direct"}, CreatedAt: oldTs})

	// New visits (after AggDateEnd) — must be counted.
	newTs := time.Now().Unix()
	dbi.Create(&model.Visits{RedirectId: id, Referer: model.Referer{Type: "social", Network: "line"}, CreatedAt: newTs})
	dbi.Create(&model.Visits{RedirectId: id, Referer: model.Referer{Type: "social", Network: "line"}, CreatedAt: newTs})

	result, err := Refresh(dbi, id)
	if err != nil {
		t.Fatalf("Refresh error: %v", err)
	}
	// 7 (base) + 2 (new) = 9; old visit not double-counted
	if result.Total != 9 {
		t.Errorf("expected total=9, got %d", result.Total)
	}
	if result.ReferrerStatistics["direct"]["all"] != 7 {
		t.Errorf("expected direct.all=7 (from base), got %d", result.ReferrerStatistics["direct"]["all"])
	}
	if result.ReferrerStatistics["social"]["all"] != 2 {
		t.Errorf("expected social.all=2 (new), got %d", result.ReferrerStatistics["social"]["all"])
	}
}

func TestRefreshUpdatesAggDateEnd(t *testing.T) {
	setupTestDB(t)
	dbi := db.Get()
	id := "refresh-aggend-01"

	before := time.Now()
	_, err := Refresh(dbi, id)
	if err != nil {
		t.Fatalf("Refresh error: %v", err)
	}

	var stat model.Statistics
	dbi.Where("redirect_id = ?", id).First(&stat)
	if !stat.AggDateEnd.After(before) && !stat.AggDateEnd.Equal(before) {
		t.Errorf("expected AggDateEnd >= %v, got %v", before, stat.AggDateEnd)
	}
}
