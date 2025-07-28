package models

import (
	"time"

	"gorm.io/datatypes"
)

type Article struct {
	ID              uint `gorm:"primaryKey"`
	Title           string
	Slug            string
	MetaTitle       string
	MetaDescription string
	FeaturedImage   string
	AltText         string
	Excerpt         string
	CanonicalURL    string
	ReadingTime     int
	Content         datatypes.JSON
	PublishedAt     *time.Time
	UpdatedAt       *time.Time
	IsFeatured      bool
	IsDeleted       bool       `gorm:"default:false"` // <-- Add this field
	Authors         []Author   `gorm:"many2many:article_authors;"`
	Categories      []Category `gorm:"many2many:article_categories;"`
	Tags            []Tag      `gorm:"many2many:article_tags;"`
	StatusID        uint
}
