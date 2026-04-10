package cleanup

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

// lastMonthTs returns a Unix timestamp from last month.
func lastMonthTs() int64 {
	now := time.Now()
	return time.Date(now.Year(), now.Month()-1, 15, 12, 0, 0, 0, time.UTC).Unix()
}

// currentMonthTs returns a Unix timestamp from the current month.
func currentMonthTs() int64 {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), 5, 12, 0, 0, 0, time.UTC).Unix()
}

func TestNextRunTimeIsFirstOfNextMonth(t *testing.T) {
	next := nextRunTime()
	now := time.Now()
	if next.Day() != 1 {
		t.Errorf("expected next run on day=1, got day=%d", next.Day())
	}
	if next.Hour() != 2 {
		t.Errorf("expected next run at hour=2, got hour=%d", next.Hour())
	}
	if !next.After(now) {
		t.Errorf("expected next run to be in the future, got %v", next)
	}
}

func TestRunCleanupNoVisits(t *testing.T) {
	setupTestDB(t)
	// Should complete without error even when there's nothing to delete.
	runCleanup()
}

func TestRunCleanupDeletesOldVisitsAndPreservesCurrentMonth(t *testing.T) {
	setupTestDB(t)
	dbi := db.Get()
	id := "cleanup-del-01"

	// Old visits (last month)
	for i := 0; i < 3; i++ {
		dbi.Create(&model.Visits{
			RedirectId: id,
			Referer:    model.Referer{Type: "direct"},
			CreatedAt:  lastMonthTs(),
		})
	}
	// Current month visits
	for i := 0; i < 2; i++ {
		dbi.Create(&model.Visits{
			RedirectId: id,
			Referer:    model.Referer{Type: "search", Network: "google"},
			CreatedAt:  currentMonthTs(),
		})
	}

	runCleanup()

	// Old visits must be gone.
	var remaining []model.Visits
	dbi.Where("redirect_id = ?", id).Find(&remaining)
	if len(remaining) != 2 {
		t.Errorf("expected 2 current-month visits to remain, got %d", len(remaining))
	}
	for _, v := range remaining {
		now := time.Now()
		currentStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Unix()
		if v.CreatedAt < currentStart {
			t.Errorf("old visit was not deleted: created_at=%d", v.CreatedAt)
		}
	}
}

func TestRunCleanupRefreshesStatisticsBeforeDeletion(t *testing.T) {
	setupTestDB(t)
	dbi := db.Get()
	id := "cleanup-stat-01"

	oldTs := lastMonthTs()
	dbi.Create(&model.Visits{RedirectId: id, Referer: model.Referer{Type: "social", Network: "facebook"}, CreatedAt: oldTs})
	dbi.Create(&model.Visits{RedirectId: id, Referer: model.Referer{Type: "direct"}, CreatedAt: oldTs})

	runCleanup()

	// Statistics must exist and reflect the visits that were deleted.
	var stat model.Statistics
	if err := dbi.Where("redirect_id = ?", id).First(&stat).Error; err != nil {
		t.Fatalf("Statistics not created before deletion: %v", err)
	}
	var result model.VisitsStatResult
	if err := json.Unmarshal([]byte(stat.Result), &result); err != nil {
		t.Fatalf("failed to parse Statistics.Result: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("expected Statistics.total=2 (preserved from deleted visits), got %d", result.Total)
	}
	if result.ReferrerStatistics["social"]["all"] != 1 {
		t.Errorf("expected social.all=1, got %d", result.ReferrerStatistics["social"]["all"])
	}
}

func TestRunCleanupPreservesStatisticsAcrossClears(t *testing.T) {
	setupTestDB(t)
	dbi := db.Get()
	id := "cleanup-preserve-01"

	// Simulate a prior Statistics record from a previous cleanup cycle.
	historicalResult := model.VisitsStatResult{
		Total:              100,
		ReferrerStatistics: map[string]map[string]int64{"direct": {"all": 100}},
		Dates:              map[string]map[string]int64{"2025-06-15": {"allday": 100}},
	}
	historicalJSON, _ := json.Marshal(historicalResult)
	// AggDateEnd is set just before last month's visits, so they will be picked up.
	aggEnd := time.Now().AddDate(0, -2, 0) // 2 months ago
	dbi.Create(&model.Statistics{
		RedirectId:   id,
		Result:       string(historicalJSON),
		AggDateStart: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		AggDateEnd:   aggEnd,
		CreatedAt:    aggEnd.Unix(),
		UpdateAt:     aggEnd.Unix(),
	})

	// Last month's visits (will be deleted by cleanup).
	dbi.Create(&model.Visits{RedirectId: id, Referer: model.Referer{Type: "search", Network: "google"}, CreatedAt: lastMonthTs()})
	dbi.Create(&model.Visits{RedirectId: id, Referer: model.Referer{Type: "search", Network: "google"}, CreatedAt: lastMonthTs()})

	runCleanup()

	// Old visits deleted.
	var remaining []model.Visits
	dbi.Where("redirect_id = ?", id).Find(&remaining)
	if len(remaining) != 0 {
		t.Errorf("expected 0 old visits remaining, got %d", len(remaining))
	}

	// Statistics must include both historical (100) and the deleted visits (2).
	var stat model.Statistics
	dbi.Where("redirect_id = ?", id).First(&stat)
	var result model.VisitsStatResult
	json.Unmarshal([]byte(stat.Result), &result)
	if result.Total != 102 {
		t.Errorf("expected Statistics.total=102 (100 historical + 2 deleted), got %d", result.Total)
	}
}
