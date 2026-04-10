package model

import "time"

// VisitsStatResult is the aggregated statistics result, shared between
// the visits handler and the statistics cache.
type VisitsStatResult struct {
	Total              int64                       `json:"total"`
	ReferrerStatistics map[string]map[string]int64 `json:"referrer_statistics"`
	Dates              map[string]map[string]int64 `json:"dates"`
}

type Statistics struct {
	Id           uint      `gorm:"primaryKey"`
	RedirectId   string    `gorm:"size:64;uniqueIndex;not null"`
	Result       string    `gorm:"type:text"`
	AggDateStart time.Time
	AggDateEnd   time.Time
	CreatedAt    int64
	UpdateAt     int64
}
