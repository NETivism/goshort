package model

import "time"

type Statistics struct {
	Id           uint   `gorm:"primaryKey"`
	RedirectId   string `gorm:"size:64,index,not null"`
	CountTotal   int64
	AggDateStart time.Time
	AggDateEnd   time.Time
	CreatedAt    int64
	UpdateAt     int64
}
