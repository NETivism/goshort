package stats

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/netivism/goshort/backend/pkg/env"
	"github.com/netivism/goshort/backend/pkg/model"
	"gorm.io/gorm"
)

// Refresh performs incremental aggregation for a given redirect:
//  1. Loads Visits newer than Statistics.AggDateEnd (or all if no Statistics exists)
//  2. Merges the delta into the existing Statistics.Result
//  3. Upserts the Statistics record
//
// Returns the merged result.
func Refresh(dbi *gorm.DB, id string) (model.VisitsStatResult, error) {
	var stat model.Statistics
	hasStat := dbi.Where("redirect_id = ?", id).First(&stat).Error == nil

	var visits []model.Visits
	if hasStat {
		dbi.Where("redirect_id = ? AND created_at > ?", id, stat.AggDateEnd.Unix()).Find(&visits)
	} else {
		dbi.Where("redirect_id = ?", id).Find(&visits)
	}

	delta := ComputeVisitStats(visits)

	var base model.VisitsStatResult
	if hasStat && stat.Result != "" {
		json.Unmarshal([]byte(stat.Result), &base)
	}
	merged := MergeVisitStats(base, delta)

	resultJSON, err := json.Marshal(merged)
	if err != nil {
		return merged, err
	}
	now := time.Now()
	if hasStat {
		dbi.Model(&stat).Updates(map[string]interface{}{
			"result":       string(resultJSON),
			"agg_date_end": now,
			"update_at":    now.Unix(),
		})
	} else {
		dbi.Create(&model.Statistics{
			RedirectId:   id,
			Result:       string(resultJSON),
			AggDateStart: now.In(env.GetTimezone()),
			AggDateEnd:   now,
			CreatedAt:    now.Unix(),
			UpdateAt:     now.Unix(),
		})
	}
	return merged, nil
}

// MergeVisitStats combines a base (historical) result with a delta (new) result.
// All counts are additive, allowing Visits to be cleared without losing history.
func MergeVisitStats(base, delta model.VisitsStatResult) model.VisitsStatResult {
	result := model.VisitsStatResult{
		Total:              base.Total + delta.Total,
		ReferrerStatistics: make(map[string]map[string]int64),
		Dates:              make(map[string]map[string]int64),
	}
	for typ, networks := range base.ReferrerStatistics {
		result.ReferrerStatistics[typ] = make(map[string]int64)
		for net, count := range networks {
			result.ReferrerStatistics[typ][net] += count
		}
	}
	for typ, networks := range delta.ReferrerStatistics {
		if result.ReferrerStatistics[typ] == nil {
			result.ReferrerStatistics[typ] = make(map[string]int64)
		}
		for net, count := range networks {
			result.ReferrerStatistics[typ][net] += count
		}
	}
	for date, hours := range base.Dates {
		result.Dates[date] = make(map[string]int64)
		for h, count := range hours {
			result.Dates[date][h] += count
		}
	}
	for date, hours := range delta.Dates {
		if result.Dates[date] == nil {
			result.Dates[date] = make(map[string]int64)
		}
		for h, count := range hours {
			result.Dates[date][h] += count
		}
	}
	return result
}

type dayData struct {
	allday int64
	hours  [24]int64
}

// ComputeVisitStats aggregates a slice of Visits into a VisitsStatResult.
func ComputeVisitStats(visits []model.Visits) model.VisitsStatResult {
	refStats := make(map[string]map[string]int64)
	dayMap := make(map[string]*dayData)

	for _, v := range visits {
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

		t := time.Unix(v.CreatedAt, 0).In(env.GetTimezone())
		dateKey := t.Format("2006-01-02")
		if dayMap[dateKey] == nil {
			dayMap[dateKey] = &dayData{}
		}
		dayMap[dateKey].allday++
		dayMap[dateKey].hours[t.Hour()]++
	}

	dates := make(map[string]map[string]int64)
	for date, data := range dayMap {
		m := map[string]int64{"allday": data.allday}
		// Always store all 24 hours so Statistics retains full precision.
		// The threshold filter is applied at the API output layer.
		for h := 0; h < 24; h++ {
			m[fmt.Sprintf("%d", h)] = data.hours[h]
		}
		dates[date] = m
	}

	return model.VisitsStatResult{
		Total:              int64(len(visits)),
		ReferrerStatistics: refStats,
		Dates:              dates,
	}
}
