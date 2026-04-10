package handle

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/netivism/goshort/backend/pkg/db"
	"github.com/netivism/goshort/backend/pkg/env"
	"github.com/netivism/goshort/backend/pkg/model"
	"github.com/netivism/goshort/backend/pkg/stats"
)

type VisitsResp struct {
	Success int                    `json:"success"`
	Message string                 `json:"message"`
	Result  model.VisitsStatResult `json:"result"`
}

func Visits(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	id := params["id"]

	dbi := db.Get()
	now := time.Now()
	forceRefresh := req.URL.Query().Get("refresh") == "1"

	// Check Statistics cache (< 1 hour), unless ?refresh=1 is set.
	var stat model.Statistics
	hasStat := dbi.Where("redirect_id = ?", id).First(&stat).Error == nil

	if !forceRefresh && hasStat && now.Unix()-stat.UpdateAt < int64(time.Hour.Seconds()) {
		var cached model.VisitsStatResult
		if err := json.Unmarshal([]byte(stat.Result), &cached); err == nil {
			writeVisitsResp(w, applyHourlyThreshold(cached), "Visits loaded from cache.")
			return
		}
	}

	// Cache miss or forced refresh: incremental refresh and upsert Statistics.
	result, _ := stats.Refresh(dbi, id)

	message := "Visits loaded successfully."
	if result.Total == 0 {
		message = "No visits found."
	}
	writeVisitsResp(w, applyHourlyThreshold(result), message)
}

// applyHourlyThreshold strips per-hour keys from dates based on VISITS_HOURLY_THRESHOLD.
// Statistics always stores all 24 hours; this filter is only applied at output time.
func applyHourlyThreshold(result model.VisitsStatResult) model.VisitsStatResult {
	threshold := env.GetVisitsHourlyThreshold()
	filtered := model.VisitsStatResult{
		Total:              result.Total,
		ReferrerStatistics: result.ReferrerStatistics,
		Dates:              make(map[string]map[string]int64),
	}
	for date, hours := range result.Dates {
		allday := hours["allday"]
		showHourly := threshold == 1 || (threshold > 1 && allday > threshold)
		m := map[string]int64{"allday": allday}
		if showHourly {
			for h := 0; h < 24; h++ {
				key := fmt.Sprintf("%d", h)
				m[key] = hours[key]
			}
		}
		filtered.Dates[date] = m
	}
	return filtered
}

func writeVisitsResp(w http.ResponseWriter, result model.VisitsStatResult, message string) {
	resp := VisitsResp{
		Success: 1,
		Message: message,
		Result:  result,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
