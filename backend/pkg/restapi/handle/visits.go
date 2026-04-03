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
)

type VisitsStatResult struct {
	Total              int64                       `json:"total"`
	ReferrerStatistics map[string]map[string]int64 `json:"referrer_statistics"`
	Dates              map[string]map[string]int64 `json:"dates"`
}

type VisitsResp struct {
	Success int              `json:"success"`
	Message string           `json:"message"`
	Result  VisitsStatResult `json:"result"`
}

func Visits(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	id := params["id"]

	dbi := db.Get()
	var visits []model.Visits
	dbi.Where("redirect_id = ?", id).Find(&visits)

	message := "Visits loaded successfully."
	if len(visits) == 0 {
		message = "No visits found."
	}

	resp := VisitsResp{
		Success: 1,
		Message: message,
		Result:  computeVisitStats(visits),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

type dayData struct {
	allday int64
	hours  [24]int64
}

func computeVisitStats(visits []model.Visits) VisitsStatResult {
	refStats := make(map[string]map[string]int64)
	dayMap := make(map[string]*dayData)

	for _, v := range visits {
		// Referrer statistics
		refType := v.Referer.Type
		if refType == "" {
			refType = "unknown"
		}
		if refStats[refType] == nil {
			refStats[refType] = make(map[string]int64)
		}
		refStats[refType]["all"]++
		if v.Referer.Network != "" {
			refStats[refType][v.Referer.Network]++
		}

		// Date statistics (in configured timezone)
		t := time.Unix(v.CreatedAt, 0).In(env.GetTimezone())
		dateKey := t.Format("2006-01-02")
		if dayMap[dateKey] == nil {
			dayMap[dateKey] = &dayData{}
		}
		dayMap[dateKey].allday++
		dayMap[dateKey].hours[t.Hour()]++
	}

	// Build dates output; include hourly breakdown based on VISITS_HOURLY_THRESHOLD.
	// 0 = disabled, 1 = always, N > 1 = only when allday > N.
	threshold := env.GetVisitsHourlyThreshold()
	dates := make(map[string]map[string]int64)
	for date, data := range dayMap {
		m := map[string]int64{"allday": data.allday}
		showHourly := threshold == 1 || (threshold > 1 && data.allday > threshold)
		if showHourly {
			for h := 0; h < 24; h++ {
				m[fmt.Sprintf("%d", h)] = data.hours[h]
			}
		}
		dates[date] = m
	}

	return VisitsStatResult{
		Total:              int64(len(visits)),
		ReferrerStatistics: refStats,
		Dates:              dates,
	}
}
