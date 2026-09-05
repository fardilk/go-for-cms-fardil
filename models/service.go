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

// Section keys the public page knows how to render. A service's Sections list
// decides which of these appear, in what order, under what heading and on what
// background — so a new service is arranged in the panel rather than in code.
const (
	SectionIntro      = "intro"
	SectionMetrics    = "metrics"
	SectionHighlights = "highlights"
	SectionReasons    = "reasons"
	SectionOutcomes   = "outcomes"
	SectionSteps      = "steps"
	SectionSchedules  = "schedules"
	SectionPlans      = "plans"
	SectionProofs     = "proofs"
	SectionFaqs       = "faqs"
	SectionCta        = "cta"
)

// ValidSectionKeys is the allow-list for a Sections entry. Anything else would
// render as nothing on the site, so it is refused at the door instead.
var ValidSectionKeys = []string{
	SectionIntro, SectionMetrics, SectionHighlights, SectionReasons, SectionOutcomes, SectionSteps,
	SectionSchedules, SectionPlans, SectionProofs, SectionFaqs, SectionCta,
}

// ValidSectionTones are the backgrounds a band may take. "auto" alternates with
// its neighbours, which is what keeps a page from running six white bands in a
// row; the rest pin a band to one colour.
var ValidSectionTones = []string{"auto", "white", "muted", "dark"}

// SectionSetting is one band on the public page.
//
// Held as JSON on the service rather than its own table: it is a short ordered
// list that is always read and written whole, never queried across services,
// and a row per band would buy nothing but joins.
type SectionSetting struct {
	Key string `json:"key"`
	// Heading override. Empty means the template's own wording, so a service
	// that is happy with the defaults stores nothing.
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Tone     string `json:"tone"`
	// A band with content can still be switched off; one without content is
	// hidden regardless, because there is nothing to show.
	Enabled bool `json:"enabled"`
}

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

	// Which bands the page renders, in order, with their headings and
	// backgrounds. Empty means "use the template's defaults", which is what
	// every service created before this existed does.
	Sections datatypes.JSON `json:"sections"`

	// One line on who the programme suits, shown as the "Cocok untuk" hint on
	// the catalogue cards and in the booking form's programme picker.
	Audience string `json:"audience"`

	// Cover for the catalogue card. Falls back to HeroImage, and to drawn
	// artwork when there is no photograph at all.
	CardImage string `json:"card_image"`

	// Participant rating. Zero means none is known, and the site shows nothing
	// rather than a fabricated score.
	RatingScore float64 `gorm:"not null;default:0" json:"rating_score"`
	RatingCount int     `gorm:"not null;default:0" json:"rating_count"`

	// Closing call to action. Empty falls back to wording built from the title.
	CtaTitle    string `json:"cta_title"`
	CtaSubtitle string `json:"cta_subtitle"`

	Highlights []ServiceHighlight `gorm:"constraint:OnDelete:CASCADE" json:"highlights"`
	Reasons    []ServiceReason    `gorm:"constraint:OnDelete:CASCADE" json:"reasons"`
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

// ServiceReason is one argument for taking the programme at all, made with a
// figure rather than an adjective: "47% reported a career advance", and under
// it where that number comes from.
//
// Link points at the write-up behind the claim. A figure a reader cannot chase
// is just a boast, so the block only shows its button when there is somewhere
// for it to go - a CMS article at /blog/<slug>, or an outside source.
type ServiceReason struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	ServiceID uint   `gorm:"not null;index" json:"service_id"`
	Position  int    `gorm:"not null;default:0" json:"position"`
	Icon      string `json:"icon"`
	// The figure itself, shown large: "47%", "20% lebih tinggi".
	Stat  string `json:"stat"`
	Title string `json:"title"`
	Body  string `json:"body"`
	// Where the figure came from: "Dari 3.340 responden".
	Source   string `json:"source"`
	LinkHref string `json:"link_href"`
	LinkText string `json:"link_text"`
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
