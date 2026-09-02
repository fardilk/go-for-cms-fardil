package models

import "time"

// Lead status values. Kept as plain strings so the panel can render them
// without a lookup table.
const (
	LeadNew       = "new"
	LeadContacted = "contacted"
	LeadWon       = "won"
	LeadLost      = "lost"

	// A registration that has been paid for and placed in a batch. It is what
	// separates someone who asked for a seat from someone who holds one.
	LeadEnrolled = "enrolled"
)

// ValidLeadStatuses is the allow-list used when the panel changes a status.
var ValidLeadStatuses = []string{LeadNew, LeadContacted, LeadWon, LeadLost, LeadEnrolled}

// What the visitor was doing. An enquiry needs a message and nothing else; a
// registration needs the details that go on a certificate and its delivery.
const (
	KindEnquiry      = "enquiry"
	KindRegistration = "registration"
)

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

	// Enquiry or registration. Older rows predate the column and read as an
	// enquiry, which is what they were.
	Kind string `gorm:"not null;default:'enquiry';index" json:"kind"`

	// Filled in by the registration form only. The name above is the one that
	// gets printed on the certificate, which is why the form asks for it in
	// full rather than reusing a nickname.
	CompanyAddress     string `json:"company_address"`
	Division           string `json:"division"`
	Position           string `json:"position"`
	City               string `json:"city"`
	CertificateAddress string `json:"certificate_address"`
	ReferralSource     string `json:"referral_source"`

	// Path of the page the form was submitted from, so it is clear which
	// service generated the enquiry.
	SourcePath string `json:"source_path"`

	Status string `gorm:"not null;default:'new';index" json:"status"`
	// Internal note, never shown on the public site.
	Note string `json:"note"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
