package routes

import (
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/fardilk/cms-porto-fardil/config"
	"github.com/fardilk/cms-porto-fardil/internal/mailer"
	"github.com/fardilk/cms-porto-fardil/internal/ratelimit"
	"github.com/fardilk/cms-porto-fardil/models"
)

// Five submissions per IP per ten minutes. Generous for a person correcting a
// typo, useless for a script.
var leadLimiter = ratelimit.New(5, 10*time.Minute)

var emailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]{2,}$`)

const (
	maxNameLen    = 120
	maxEmailLen   = 200
	maxPhoneLen   = 40
	maxCompanyLen = 160
	maxMessageLen = 4000
	maxSourceLen  = 300
	maxAddressLen = 400
	maxShortLen   = 160
)

type leadRequest struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Phone   string `json:"phone"`
	Company string `json:"company"`
	Message string `json:"message"`

	SourcePath string `json:"source_path"`

	Kind string `json:"kind"`

	CompanyAddress     string `json:"company_address"`
	Division           string `json:"division"`
	Position           string `json:"position"`
	City               string `json:"city"`
	CertificateAddress string `json:"certificate_address"`
	ReferralSource     string `json:"referral_source"`

	ProgramCategory  string `json:"program_category"`
	ProgramSlug      string `json:"program_slug"`
	Participants     string `json:"participants"`
	DeliveryMode     string `json:"delivery_mode"`
	PreferredBatch   string `json:"preferred_batch"`
	BudgetRange      string `json:"budget_range"`
	PreferredContact string `json:"preferred_contact"`

	// Website is a honeypot. It is hidden from people and left empty by them;
	// anything that fills it is automated.
	Website string `json:"website"`
}

func trimTo(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) > n {
		return s[:n]
	}
	return s
}

// CreateLead handles POST /api/leads. This is the only unauthenticated write
// on the API, so it validates, rate limits and traps bots.
func CreateLead(c *gin.Context) {
	if allowed, retry := leadLimiter.Allow(c.ClientIP()); !allowed {
		c.Header("Retry-After", strconv.Itoa(int(retry.Seconds())+1))
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error": "Terlalu banyak pengiriman. Coba lagi beberapa menit lagi.",
		})
		return
	}

	var req leadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format permintaan tidak valid"})
		return
	}

	// Report success to a bot so it has nothing to learn from, and store
	// nothing.
	if strings.TrimSpace(req.Website) != "" {
		c.JSON(http.StatusCreated, gin.H{"ok": true})
		return
	}

	// Anything the site does not know how to send is filed as an enquiry, which
	// is the kind with the loosest requirements and so never loses a message.
	kind := models.KindEnquiry
	if slices.Contains(models.ValidLeadKinds, req.Kind) {
		kind = req.Kind
	}

	lead := models.Lead{
		Name:       trimTo(req.Name, maxNameLen),
		Email:      trimTo(req.Email, maxEmailLen),
		Phone:      trimTo(req.Phone, maxPhoneLen),
		Company:    trimTo(req.Company, maxCompanyLen),
		Message:    trimTo(req.Message, maxMessageLen),
		SourcePath: trimTo(req.SourcePath, maxSourceLen),
		Kind:       kind,
		Status:     models.LeadNew,

		CompanyAddress:     trimTo(req.CompanyAddress, maxAddressLen),
		Division:           trimTo(req.Division, maxShortLen),
		Position:           trimTo(req.Position, maxShortLen),
		City:               trimTo(req.City, maxShortLen),
		CertificateAddress: trimTo(req.CertificateAddress, maxAddressLen),
		ReferralSource:     trimTo(req.ReferralSource, maxShortLen),

		ProgramCategory:  trimTo(req.ProgramCategory, maxShortLen),
		ProgramSlug:      trimTo(req.ProgramSlug, maxShortLen),
		Participants:     trimTo(req.Participants, maxShortLen),
		DeliveryMode:     trimTo(req.DeliveryMode, maxShortLen),
		PreferredBatch:   trimTo(req.PreferredBatch, maxShortLen),
		BudgetRange:      trimTo(req.BudgetRange, maxShortLen),
		PreferredContact: trimTo(req.PreferredContact, maxShortLen),
	}

	if fields := missingLeadFields(lead); len(fields) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "Mohon lengkapi data berikut",
			"fields": fields,
		})
		return
	}

	if err := config.DB.Create(&lead).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan pesan"})
		return
	}

	notifyNewLead(lead)

	c.JSON(http.StatusCreated, gin.H{"ok": true, "id": lead.ID})
}

func missingLeadFields(l models.Lead) map[string]string {
	fields := map[string]string{}
	if l.Name == "" {
		fields["name"] = "Nama wajib diisi"
	}
	if l.Email == "" {
		fields["email"] = "Email wajib diisi"
	} else if !emailPattern.MatchString(l.Email) {
		fields["email"] = "Format email tidak valid"
	}
	if l.Phone == "" {
		fields["phone"] = "Nomor telepon wajib diisi"
	}

	// Each kind asks for the least it can act on. Requiring a certificate
	// address to book a discovery call, or a written brief to reserve a named
	// seat, only costs submissions.
	var required map[string]string

	switch l.Kind {
	case models.KindRegistration:
		// A registration is a commitment to print a certificate and post it, so
		// every field that decides what is printed and where it goes is required.
		required = map[string]string{
			"company":             ifBlank(l.Company, "Instansi/perusahaan wajib diisi"),
			"company_address":     ifBlank(l.CompanyAddress, "Alamat perusahaan wajib diisi"),
			"division":            ifBlank(l.Division, "Divisi/departemen wajib diisi"),
			"position":            ifBlank(l.Position, "Jabatan wajib diisi"),
			"city":                ifBlank(l.City, "Kota domisili wajib diisi"),
			"certificate_address": ifBlank(l.CertificateAddress, "Alamat pengiriman sertifikat wajib diisi"),
			"referral_source":     ifBlank(l.ReferralSource, "Mohon pilih dari mana Anda mendapat info"),
		}

	case models.KindConsultation:
		// Nothing is being booked yet. We need to know who they are, what area
		// they are asking about, what the problem is, and how to reach them.
		required = map[string]string{
			"company":           ifBlank(l.Company, "Instansi/perusahaan wajib diisi"),
			"position":          ifBlank(l.Position, "Jabatan wajib diisi"),
			"program_category":  ifBlank(l.ProgramCategory, "Pilih bidang yang ingin dibahas"),
			"message":           ifBlank(l.Message, "Ceritakan kebutuhan atau tantangan Anda"),
			"preferred_contact": ifBlank(l.PreferredContact, "Pilih cara kami menghubungi Anda"),
		}

	case models.KindReservation:
		// A seat is being held in a named programme, so the programme itself is
		// the one thing that cannot be missing.
		required = map[string]string{
			"company":          ifBlank(l.Company, "Instansi/perusahaan wajib diisi"),
			"program_category": ifBlank(l.ProgramCategory, "Pilih kategori program"),
			"program_slug":     ifBlank(l.ProgramSlug, "Pilih program yang ingin direservasi"),
			"participants":     ifBlank(l.Participants, "Isi jumlah peserta"),
			"delivery_mode":    ifBlank(l.DeliveryMode, "Pilih format pelaksanaan"),
		}

	default:
		// An enquiry is nothing without the question.
		if l.Message == "" {
			fields["message"] = "Pesan wajib diisi"
		}
		return fields
	}

	for name, message := range required {
		if message != "" {
			fields[name] = message
		}
	}
	return fields
}

// ifBlank returns the message when the value is empty, and "" when it is not,
// so a required-field table reads as one line per field.
func ifBlank(value, message string) string {
	if strings.TrimSpace(value) == "" {
		return message
	}
	return ""
}

func notifyNewLead(l models.Lead) {
	switch l.Kind {
	case models.KindRegistration:
		notifyNewRegistration(l)
		return
	case models.KindConsultation:
		notifyNewConsultation(l)
		return
	case models.KindReservation:
		notifyNewReservation(l)
		return
	}

	company := l.Company
	if company == "" {
		company = "-"
	}
	source := l.SourcePath
	if source == "" {
		source = "-"
	}

	body := fmt.Sprintf(
		"Pesan baru dari situs Excellence Plus Indonesia.\n\n"+
			"Nama       : %s\n"+
			"Email      : %s\n"+
			"Telepon    : %s\n"+
			"Perusahaan : %s\n"+
			"Halaman    : %s\n"+
			"Waktu      : %s\n\n"+
			"Pesan:\n%s\n",
		l.Name, l.Email, l.Phone, company, source,
		l.CreatedAt.Format("2 January 2006 15:04"), l.Message,
	)

	mailer.SendAsync(mailer.FromEnv(), "Lead baru: "+l.Name, body)
}

func notifyNewRegistration(l models.Lead) {
	body := fmt.Sprintf(
		"Pendaftaran baru dari situs Excellence Plus Indonesia.\n\n"+
			"Nama (sertifikat) : %s\n"+
			"Email             : %s\n"+
			"Telepon           : %s\n"+
			"Instansi          : %s\n"+
			"Alamat instansi   : %s\n"+
			"Divisi            : %s\n"+
			"Jabatan           : %s\n"+
			"Kota domisili     : %s\n"+
			"Kirim sertifikat  : %s\n"+
			"Info dari         : %s\n"+
			"Halaman           : %s\n"+
			"Waktu             : %s\n",
		l.Name, l.Email, l.Phone, l.Company, l.CompanyAddress, l.Division,
		l.Position, l.City, l.CertificateAddress, l.ReferralSource, l.SourcePath,
		l.CreatedAt.Format("2 January 2006 15:04"),
	)
	if strings.TrimSpace(l.Message) != "" {
		body += "\nCatatan:\n" + l.Message + "\n"
	}

	mailer.SendAsync(mailer.FromEnv(), "Pendaftaran baru: "+l.Name, body)
}

// dash keeps an optional answer from printing as an empty line in the email.
func dash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

func notifyNewConsultation(l models.Lead) {
	body := fmt.Sprintf(
		"Permintaan konsultasi dari situs Excellence Plus Indonesia.\n\n"+
			"Nama           : %s\n"+
			"Email          : %s\n"+
			"Telepon        : %s\n"+
			"Instansi       : %s\n"+
			"Jabatan        : %s\n"+
			"Bidang         : %s\n"+
			"Jumlah peserta : %s\n"+
			"Format         : %s\n"+
			"Target waktu   : %s\n"+
			"Kisaran budget : %s\n"+
			"Dihubungi via  : %s\n"+
			"Halaman        : %s\n"+
			"Waktu          : %s\n\n"+
			"Kebutuhan:\n%s\n",
		l.Name, l.Email, l.Phone, dash(l.Company), dash(l.Position),
		dash(l.ProgramCategory), dash(l.Participants), dash(l.DeliveryMode),
		dash(l.PreferredBatch), dash(l.BudgetRange), dash(l.PreferredContact),
		dash(l.SourcePath), l.CreatedAt.Format("2 January 2006 15:04"), l.Message,
	)

	mailer.SendAsync(mailer.FromEnv(), "Konsultasi baru: "+l.Name, body)
}

func notifyNewReservation(l models.Lead) {
	body := fmt.Sprintf(
		"Reservasi program dari situs Excellence Plus Indonesia.\n\n"+
			"Nama           : %s\n"+
			"Email          : %s\n"+
			"Telepon        : %s\n"+
			"Instansi       : %s\n"+
			"Jabatan        : %s\n"+
			"Kategori       : %s\n"+
			"Program        : %s\n"+
			"Jumlah peserta : %s\n"+
			"Format         : %s\n"+
			"Batch/waktu    : %s\n"+
			"Dihubungi via  : %s\n"+
			"Info dari      : %s\n"+
			"Halaman        : %s\n"+
			"Waktu          : %s\n",
		l.Name, l.Email, l.Phone, dash(l.Company), dash(l.Position),
		dash(l.ProgramCategory), dash(l.ProgramSlug), dash(l.Participants),
		dash(l.DeliveryMode), dash(l.PreferredBatch), dash(l.PreferredContact),
		dash(l.ReferralSource), dash(l.SourcePath),
		l.CreatedAt.Format("2 January 2006 15:04"),
	)
	if strings.TrimSpace(l.Message) != "" {
		body += "\nCatatan:\n" + l.Message + "\n"
	}

	mailer.SendAsync(mailer.FromEnv(), "Reservasi baru: "+l.Name, body)
}

// GetLeads handles GET /api/leads. Returns the page plus the count of unread
// leads, so the panel can render its sidebar badge without a second request.
func GetLeads(c *gin.Context) {
	// The panel splits the same table by what the visitor was doing: enquiries,
	// consultation requests, seats still being processed, and people who hold
	// one. Applied to a fresh query each time it is needed, so counting cannot
	// alter the query the page itself is read from.
	scope := func() *gorm.DB {
		tx := config.DB.Model(&models.Lead{})
		if kind := c.Query("kind"); slices.Contains(models.ValidLeadKinds, kind) {
			tx = tx.Where("kind = ?", kind)
		}

		// A reservation and a registration are the same thing at different
		// stages of completeness, so Transaksi counts both.
		seatKinds := []string{models.KindRegistration, models.KindReservation}
		switch c.Query("stage") {
		case "permintaan":
			tx = tx.Where("kind IN ?", seatKinds).
				Where("status IN ?", []string{models.LeadNew, models.LeadContacted})
		case "peserta":
			tx = tx.Where("kind IN ?", seatKinds).
				Where("status = ?", models.LeadEnrolled)
		}
		return tx
	}

	tx := scope()
	if status := c.Query("status"); status != "" && slices.Contains(models.ValidLeadStatuses, status) {
		tx = tx.Where("status = ?", status)
	}

	var total int64
	tx.Count(&total)

	limit := 50
	if v, err := strconv.Atoi(c.Query("limit")); err == nil && v > 0 && v <= 200 {
		limit = v
	}
	offset := 0
	if v, err := strconv.Atoi(c.Query("offset")); err == nil && v > 0 {
		offset = v
	}

	var leads []models.Lead
	if err := tx.Order("created_at DESC").Limit(limit).Offset(offset).Find(&leads).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch leads"})
		return
	}

	// Unread within this list. Counted globally, the badge on one page would
	// report the backlog of another.
	var newCount int64
	scope().Where("status = ?", models.LeadNew).Count(&newCount)

	c.JSON(http.StatusOK, gin.H{
		"items":     leads,
		"total":     total,
		"new_count": newCount,
	})
}

// UpdateLead handles PATCH /api/leads/:id for the status and the internal note.
// Nothing the visitor submitted is editable; this is a record of what they sent.
func UpdateLead(c *gin.Context) {
	var lead models.Lead
	if err := config.DB.First(&lead, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Lead not found"})
		return
	}

	var body struct {
		Status *string `json:"status"`
		Note   *string `json:"note"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if body.Status != nil {
		if !slices.Contains(models.ValidLeadStatuses, *body.Status) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "status must be one of: " + strings.Join(models.ValidLeadStatuses, ", "),
			})
			return
		}
		lead.Status = *body.Status
	}
	if body.Note != nil {
		lead.Note = trimTo(*body.Note, maxMessageLen)
	}

	if err := config.DB.Save(&lead).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update lead"})
		return
	}

	c.JSON(http.StatusOK, lead)
}

// DeleteLead handles DELETE /api/leads/:id, for clearing spam that slipped past
// the honeypot.
func DeleteLead(c *gin.Context) {
	if err := config.DB.Delete(&models.Lead{}, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete lead"})
		return
	}
	c.Status(http.StatusNoContent)
}
