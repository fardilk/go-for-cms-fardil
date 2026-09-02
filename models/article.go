package models

import (
	"time"

	"gorm.io/datatypes"
)

// Article is one blog post.
//
// The JSON tags matter: without them Go emits Go field names, so the API
// answered with `Title` while every consumer looked for `title`. The panel's
// list filtered on the lowercase key and therefore discarded every article it
// was given, which made the list look permanently empty.
type Article struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	Title           string         `json:"title"`
	Slug            string         `gorm:"index" json:"slug"`
	MetaTitle       string         `json:"meta_title"`
	MetaDescription string         `json:"meta_description"`
	FeaturedImage   string         `json:"featured_image"`
	AltText         string         `json:"alt_text"`
	Excerpt         string         `json:"excerpt"`
	CanonicalURL    string         `json:"canonical_url"`
	ReadingTime     int            `json:"reading_time"`
	Content         datatypes.JSON `json:"content"`
	PublishedAt     *time.Time     `json:"published_at"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       *time.Time     `json:"updated_at"`
	IsFeatured      bool           `json:"is_featured"`

	// Soft delete. GetArticles filters on it; a trashed article used to keep
	// appearing in the list because nothing did.
	IsDeleted bool `gorm:"default:false;index" json:"is_deleted"`

	Authors    []Author   `gorm:"many2many:article_authors;" json:"authors"`
	Categories []Category `gorm:"many2many:article_categories;" json:"categories"`
	Tags       []Tag      `gorm:"many2many:article_tags;" json:"tags"`

	StatusID uint `json:"status_id"`
	// Named apart from "status" on purpose: callers have long sent that key as a
	// plain string, and binding it into a struct turns their request into a 400.
	Status *Status `json:"status_detail,omitempty"`
}
