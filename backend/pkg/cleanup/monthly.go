package cleanup

import (
	"log"
	"time"

	"github.com/netivism/goshort/backend/pkg/db"
	"github.com/netivism/goshort/backend/pkg/env"
	"github.com/netivism/goshort/backend/pkg/model"
	"github.com/netivism/goshort/backend/pkg/stats"
)

// StartMonthlyCleanup launches a background goroutine that runs on the 1st of
// every month at 02:00 (in the configured timezone).
//
// Each run:
//  1. Collects all distinct redirect IDs that have Visits older than the current month.
//  2. Calls stats.Refresh for each, ensuring Statistics is fully up-to-date.
//  3. Deletes all Visits records created before the current month start.
func StartMonthlyCleanup() {
	go func() {
		for {
			next := nextRunTime()
			log.Printf("[cleanup] Next monthly visits cleanup scheduled at %s", next.Format(time.RFC3339))
			time.Sleep(time.Until(next))
			runCleanup()
		}
	}()
}

// nextRunTime returns 02:00 on the 1st of next month in the configured timezone.
func nextRunTime() time.Time {
	tz := env.GetTimezone()
	now := time.Now().In(tz)
	// First day of next month at 02:00
	firstOfNext := time.Date(now.Year(), now.Month()+1, 1, 2, 0, 0, 0, tz)
	return firstOfNext
}

func runCleanup() {
	tz := env.GetTimezone()
	now := time.Now().In(tz)
	// Start of the current month – everything before this will be deleted.
	currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, tz)
	cutoff := currentMonthStart.Unix()

	log.Printf("[cleanup] Starting monthly visits cleanup (deleting visits before %s)", currentMonthStart.Format("2006-01-02"))

	dbi := db.Get()

	// Collect distinct redirect IDs that have visits we're about to delete.
	var redirectIDs []string
	dbi.Model(&model.Visits{}).
		Where("created_at < ?", cutoff).
		Distinct("redirect_id").
		Pluck("redirect_id", &redirectIDs)

	if len(redirectIDs) == 0 {
		log.Println("[cleanup] No visits to clean up.")
		return
	}

	log.Printf("[cleanup] Refreshing statistics for %d redirect(s) before deletion...", len(redirectIDs))

	failed := 0
	for _, id := range redirectIDs {
		if _, err := stats.Refresh(dbi, id); err != nil {
			log.Printf("[cleanup] Failed to refresh statistics for redirect %s: %v", id, err)
			failed++
		}
	}

	if failed > 0 {
		log.Printf("[cleanup] %d redirect(s) failed statistics refresh; aborting deletion to prevent data loss.", failed)
		return
	}

	// All statistics refreshed – safe to delete old visits.
	result := dbi.Where("created_at < ?", cutoff).Delete(&model.Visits{})
	if result.Error != nil {
		log.Printf("[cleanup] Error deleting old visits: %v", result.Error)
		return
	}
	log.Printf("[cleanup] Deleted %d visit record(s) older than %s.", result.RowsAffected, currentMonthStart.Format("2006-01-02"))
}
