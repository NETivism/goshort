package model

type Redirect struct {
	Id        string `gorm:"size:64,primaryKey"`
	Redirect  string `gorm:"size:1024,not null"`
	Domain    string `gorm:"index"`
	Path      string `gorm:"index"`
	CreatedAt int64
	UpdateAt  int64
}
