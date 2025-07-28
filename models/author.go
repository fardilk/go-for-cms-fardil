package models

type Author struct {
	ID     uint `gorm:"primaryKey"`
	Name   string
	Bio    string
	Avatar string
	Email  string
	UserID uint `gorm:"uniqueIndex"` // One-to-one relation
	User   User // Foreign key relation
}
