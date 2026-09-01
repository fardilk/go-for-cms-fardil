package models

import (
	"time"

	"gorm.io/datatypes"
)

// Template decides which sections a service page renders and what the CMS
// labels its repeating groups. The shapes are deliberately shared: a training
// module, a consulting phase and an SLA stage are all "ordered step with a
// short duration", so they use one table and one editor with different labels.
const (
	TemplateProgram    = "program"    // training and coaching: scheduled, priced, enrollable
	TemplateEngagement = "engagement" // consultancy: scoped project, quoted
	TemplateRetainer   = "retainer"   // executive search and EOR: ongoing service with an SLA
)

// ValidTemplates is the allow-list used when validating writes.
var ValidTemplates = []string{TemplateProgram, TemplateEngagement, TemplateRetainer}

// Service is one page under /services/:category/:slug.
type Service struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Slug     string `gorm:"not null;uniqueIndex:idx_service_path" json:"slug"`
	Category string `gorm:"not null;uniqueIndex:idx_service_path" json:"category"`

	CategoryLabel string `json:"category_label"`
	Template      string `gorm:"not null;index" json:"template"`
	Title         string `gorm:"not null" json:"title"`
	Subtitle      string `json:"subtitle"`

	// Draft pages are editable in the CMS but excluded from the public feed the
	// site builds from.
	Published bool `gorm:"not null;default:false;index" json:"published"`
	SortOrder int  `gorm:"not null;default:0" json:"sort_order"`

	MetaTitle       string `json:"meta_title"`
	MetaDescription string `json:"meta_description"`
	CanonicalURL    string `json:"canonical_url"`
	OgImage         string `json:"og_image"`

	HeroEyebrow      string `json:"hero_eyebrow"`
	HeroHeadline     string `json:"hero_headline"`
	HeroSubheadline  string `json:"hero_subheadline"`
	HeroImage        string `json:"hero_image"`
	PrimaryCtaText   string `json:"primary_cta_text"`
	PrimaryCtaHref   string `json:"primary_cta_href"`
	SecondaryCtaText string `json:"secondary_cta_text"`
	SecondaryCtaHref string `json:"secondary_cta_href"`

	// Free-text intro rendered above the first repeating group.
	Intro string `json:"intro"`

	Highlights []ServiceHighlight `gorm:"constraint:OnDelete:CASCADE" json:"highlights"`
	Steps      []ServiceStep      `gorm:"constraint:OnDelete:CASCADE" json:"steps"`
	Outcomes   []ServiceOutcome   `gorm:"constraint:OnDelete:CASCADE" json:"outcomes"`
	Metrics    []ServiceMetric    `gorm:"constraint:OnDelete:CASCADE" json:"metrics"`
	Faqs       []ServiceFaq       `gorm:"constraint:OnDelete:CASCADE" json:"faqs"`
	Plans      []ServicePlan      `gorm:"constraint:OnDelete:CASCADE" json:"plans"`
	Proofs     []ServiceProof     `gorm:"constraint:OnDelete:CASCADE" json:"proofs"`
	Schedules  []ServiceSchedule  `gorm:"constraint:OnDelete:CASCADE" json:"schedules"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ServiceHighlight is "who this is for" (program), "symptoms you recognise"
// (engagement) or "what we cover" (retainer).
type ServiceHighlight struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	ServiceID uint   `gorm:"not null;index" json:"service_id"`
	Position  int    `gorm:"not null;default:0" json:"position"`
	Icon      string `json:"icon"`
	Title     string `json:"title"`
	Body      string `json:"body"`
}

// ServiceStep is a curriculum module, a delivery phase or an SLA stage. Meta
// carries the short qualifier the template shows beside the title: "4 jam",
// "Minggu 1-2", "3 hari kerja".
type ServiceStep struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	ServiceID uint   `gorm:"not null;index" json:"service_id"`
	Position  int    `gorm:"not null;default:0" json:"position"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Meta      string `json:"meta"`
}

// ServiceOutcome is a single bullet: a learning outcome, a deliverable or a
// compliance item.
type ServiceOutcome struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	ServiceID uint   `gorm:"not null;index" json:"service_id"`
	Position  int    `gorm:"not null;default:0" json:"position"`
	Icon      string `json:"icon"`
	Text      string `json:"text"`
}

type ServiceMetric struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	ServiceID uint   `gorm:"not null;index" json:"service_id"`
	Position  int    `gorm:"not null;default:0" json:"position"`
	Label     string `json:"label"`
	Value     string `json:"value"`
}

type ServiceFaq struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	ServiceID uint   `gorm:"not null;index" json:"service_id"`
	Position  int    `gorm:"not null;default:0" json:"position"`
	Question  string `json:"question"`
	Answer    string `json:"answer"`
}

// ServicePlan is a price column. Features is a JSON array of strings so the
// editor can add rows without another table.
type ServicePlan struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	ServiceID   uint           `gorm:"not null;index" json:"service_id"`
	Position    int            `gorm:"not null;default:0" json:"position"`
	Name        string         `json:"name"`
	Price       string         `json:"price"`
	Note        string         `json:"note"`
	Highlighted bool           `gorm:"not null;default:false" json:"highlighted"`
	Features    datatypes.JSON `json:"features"`
}

// ServiceProof is a testimonial or a case study; Kind says which.
type ServiceProof struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	ServiceID uint   `gorm:"not null;index" json:"service_id"`
	Position  int    `gorm:"not null;default:0" json:"position"`
	Kind      string `gorm:"not null;default:'testimonial'" json:"kind"` // testimonial | case
	Name      string `json:"name"`
	Role      string `json:"role"`
	Company   string `json:"company"`
	Quote     string `json:"quote"`
	Result    string `json:"result"`
	Image     string `json:"image"`
}

// ServiceSchedule is one open batch. Only the program template renders these.
type ServiceSchedule struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	ServiceID   uint       `gorm:"not null;index" json:"service_id"`
	Position    int        `gorm:"not null;default:0" json:"position"`
	StartsAt    *time.Time `json:"starts_at"`
	EndsAt      *time.Time `json:"ends_at"`
	City        string     `json:"city"`
	Format      string     `json:"format"` // public | in-house | online
	SeatsTotal  int        `json:"seats_total"`
	SeatsLeft   int        `json:"seats_left"`
	Price       string     `json:"price"`
	RegisterURL string     `json:"register_url"`
}
