package model

type Visits struct {
	Id         uint    `gorm:"primaryKey"`
	RedirectId string  `gorm:"size:64;index;not null"`
	Referer    Referer `gorm:"embedded;embeddedPrefix:referer_"`
	CreatedAt  int64
}

type Referer struct {
	Type    string
	Network string
	Link    string
}
