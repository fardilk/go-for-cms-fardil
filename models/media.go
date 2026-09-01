package models

import "time"

// MediaAsset records one uploaded file. The bytes live on disk under the
// upload directory; this table holds what the CMS needs to list and pick them.
type MediaAsset struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Filename     string    `gorm:"not null;uniqueIndex" json:"filename"`
	OriginalName string    `json:"original_name"`
	MimeType     string    `json:"mime_type"`
	SizeBytes    int64     `json:"size_bytes"`
	Width        int       `json:"width"`
	Height       int       `json:"height"`
	AltText      string    `json:"alt_text"`
	CreatedAt    time.Time `json:"created_at"`
}

// URL is the public path the site and the CMS use to reference the file.
func (m MediaAsset) URL() string {
	return "/images/" + m.Filename
}
