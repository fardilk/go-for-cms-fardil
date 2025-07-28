package models

import "time"

// Tag represents an article tag.
type Tag struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"unique;not null"`
	Slug      string `gorm:"unique;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
