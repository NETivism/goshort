package model

type Visits struct {
	Id         uint    `gorm:"primaryKey"`
	RedirectId string  `gorm:"size:64,index"`
	Utm        Utm     `gorm:"embedded;embeddedPrefix:utm_"`
	Referer    Referer `gorm:"embedded;embeddedPrefix:referer_"`
	CreatedAt  int64
}

type Utm struct {
	Source   string
	Medium   string
	Term     string
	Content  string
	Campaign string
}

type Referer struct {
	Type    string
	Network string
	Link    string
}
