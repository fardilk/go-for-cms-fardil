package models

import "time"

// Lead status values. Kept as plain strings so the panel can render them
// without a lookup table.
const (
	LeadNew       = "new"
	LeadContacted = "contacted"
	LeadWon       = "won"
	LeadLost      = "lost"
)

// ValidLeadStatuses is the allow-list used when the panel changes a status.
var ValidLeadStatuses = []string{LeadNew, LeadContacted, LeadWon, LeadLost}

// Lead is one enquiry submitted from the public site.
//
// No IP address or user agent is stored. Rate limiting works without keeping
// them, and holding them would add a data-protection obligation for no
// operational gain.
type Lead struct {
	ID uint `gorm:"primaryKey" json:"id"`

	Name    string `gorm:"not null" json:"name"`
	Email   string `gorm:"not null;index" json:"email"`
	Phone   string `gorm:"not null" json:"phone"`
	Company string `json:"company"`
	Message string `gorm:"not null" json:"message"`

	// Path of the page the form was submitted from, so it is clear which
	// service generated the enquiry.
	SourcePath string `json:"source_path"`

	Status string `gorm:"not null;default:'new';index" json:"status"`
	// Internal note, never shown on the public site.
	Note string `json:"note"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
